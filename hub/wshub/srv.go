// Package wshub provides a websocket server and client using gorilla/websocket for package hub.
package wshub

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"xelf.org/daql/hub"
	"xelf.org/daql/log"
)

type Server struct {
	*hub.Hub
	websocket.Upgrader
	Log log.Logger
}

func NewServer(h *hub.Hub) *Server { return &Server{Hub: h, Log: log.Root} }

func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	wc, err := s.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Error("wshub upgrade failed", "err", err)
		return
	}
	route := s.Hub.Chan()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	user, _ := r.Context().Value("user").(string)
	c := newConn(hub.NextID(), wc, nil, user)
	c.tick = t.C
	route <- &hub.Msg{From: s.Hub, Subj: hub.Signon}
	go c.writeAll(0, s.Log)
	err = c.readAll(route)
	route <- &hub.Msg{From: s.Hub, Subj: hub.Signoff}
	if err != nil {
		s.Log.Error("wshub read failed", "err", err)
	}
}
