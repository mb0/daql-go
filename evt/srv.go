package evt

import (
	"fmt"
	"time"

	"xelf.org/daql/hub"
)

// Server provides hub services to subscribe and publish to a ledger.
type Server struct {
	Publisher
	*Ctrl
}

func NewServer(pubr Publisher) *Server { return &Server{pubr, NewCtrl(pubr)} }

func (srv *Server) Router() hub.Router {
	return hub.NewPrefixFilter(hub.RouterFunc(func(m *hub.Msg) {
		srv.Input <- m
	}), hub.Signoff, "evt.")
}

func (srv *Server) Services() hub.Services {
	return hub.Services{
		"evt.pub": PubFunc(srv.pub),
		"evt.sat": SatFunc(srv.sat),
	}.Merge(srv.Ctrl.Services())
}

func (srv *Server) Run() {
	s := srv.Ctrl.Services()
	for m := range srv.Input {
		if !s.Handle(m) {
			srv.Ctrl.Handle(m)
		}
	}
}
func (srv *Server) sat(m *hub.Msg, req SatReq) (*Update, error) {
	if len(req.Trans) > 0 { // we have offline transactions
		var all []*Event
		for _, t := range req.Trans {
			t.Audit.Arrived = time.Now()
			_, res, err := srv.Publish(t)
			if err != nil {
				return nil, err
			}
			all = append(all, res...)
		}
		_, trig := srv.Subs.Show(m.From, all)
		if trig { // trigger if other subscribers or any monitors need to be sent
			srv.Btrig()
		}
	}
	// call ctrl sub for subscription and up updates
	return srv.Ctrl.sub(m, SubReq{Rev: req.Rev, Tops: req.Tops})
}

func (srv *Server) pub(m *hub.Msg, req PubReq) (*Update, error) {
	if len(req.Acts) == 0 {
		return nil, fmt.Errorf("no actions")
	}
	if req.User == "" {
		req.User = m.From.User()
	}
	oldrev := srv.Rev()
	req.Trans.Arrived = time.Now()
	rev, evs, err := srv.Publish(req.Trans)
	if err != nil {
		return nil, err
	}
	return showSubs(srv.Ctrl, m.From, oldrev, rev, evs), nil
}

func showSubs(ctrl *Ctrl, from hub.Conn, old, rev time.Time, evs []*Event) *Update {
	// show events to all subscribers except the sender, monitors are processed normally
	s, trig := ctrl.Subs.Show(from, evs)
	if trig { // trigger if other subscribers or any monitors need to be sent
		ctrl.Btrig()
	}
	if s.Note || len(s.Bufr) > 0 {
		// flush all update to sender before returning with new results
		s.Chan() <- &hub.Msg{Subj: "evt.update", Data: s.Update(old)}
	}
	if rev.After(s.Rev) {
		s.Rev = rev
	}
	// send all events to the sender, independet of any subscriptions
	return &Update{Rev: rev, Evs: evs}
}
