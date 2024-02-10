// Package wshub provides a websocket server and client using gorilla/websocket for package hub.
package wshub

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"xelf.org/daql/hub"
	"xelf.org/daql/log"
)

type Server struct {
	*hub.Hub
	websocket.Upgrader
	Timeout  time.Duration
	UserFunc func(*http.Request) (string, error)
	Log      log.Logger
}

func NewServer(h *hub.Hub) *Server { return &Server{Hub: h, Log: log.Root} }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var user string
	if s.UserFunc != nil {
		u, err := s.UserFunc(r)
		if err != nil {
			s.Log.Error("wshub user func failed", "err", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		user = u
	}
	wc, err := s.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Error("wshub upgrade failed", "err", err)
		// the upgrader already writes a http error if appropriate
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c := newConn(ctx, hub.NextID(), wc, nil, user)
	if s.Timeout == 0 {
		s.Timeout = time.Minute
	}
	t := time.NewTicker(s.Timeout)
	defer t.Stop()
	c.tick = t.C
	route := s.Hub.Chan()
	route <- &hub.Msg{From: c, Subj: hub.Signon}
	go c.writeAll(0, s.Log, s.Timeout)
	err = c.readAll(route)
	route <- &hub.Msg{From: c, Subj: hub.Signoff}
	if err != nil {
		s.Log.Error("wshub read failed", "err", err)
	}
}
