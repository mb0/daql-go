package evt

import (
	"encoding/json"
	"fmt"
	"time"

	"xelf.org/daql/hub"
)

type ModelAuthority = func(evs []Action) bool

// Satellite connects to a server hub, replicates events, and manages local subscriptions.
// Satellites can publish authoritative events locally to support offline use to some extent.
type Satellite struct {
	LocalPublisher
	*Ctrl
	cli    hub.Conn
	auth   ModelAuthority
	tops   []string
	toks   hub.TokMap
	remote chan *hub.Msg
	status Status
}

func New(rep LocalPublisher, cli hub.Conn, auth ModelAuthority) *Satellite {
	pr := rep.Project()
	tops := make([]string, 0, len(pr.Schemas)*8)
	for _, s := range pr.Schemas {
		for _, m := range s.Models {
			tops = append(tops, m.Qualified())
		}
	}
	return &Satellite{LocalPublisher: rep, Ctrl: NewCtrl(rep), cli: cli, tops: tops,
		remote: make(chan *hub.Msg, 64),
	}
}

func (sat *Satellite) Router() hub.Router {
	return hub.NewMatchFilter(hub.RouterFunc(func(m *hub.Msg) {
		sat.Input <- m
	}), "_signoff", "evt.")
}
func (sat *Satellite) CliRouter() hub.Router {
	return hub.NewPrefixFilter(hub.RouterFunc(func(m *hub.Msg) {
		sat.remote <- m
	}), "_sign", "evt.")
}

func (sat *Satellite) Services() hub.Services {
	return hub.Services{
		"evt.pub": PubFunc(sat.pub),
	}.Merge(sat.Ctrl.Services())
}
func (sat *Satellite) Run() {
	s := sat.Services()
	for {
		select {
		case m := <-sat.Input:
			if !s.Handle(m) {
				sat.Ctrl.Handle(m)
			}
		case m := <-sat.remote:
			sat.handleRemote(m)
		}
	}
}

func (sat *Satellite) stat(m *hub.Msg) (*Status, error) {
	s := sat.status
	if m.From == nil {
		sat.Subs.BcastMsg(&hub.Msg{Subj: m.Subj, Data: s})
	}
	return &s, nil
}
func (sat *Satellite) isOnline() bool              { return sat.status.Off.IsZero() && !sat.status.On.IsZero() }
func (sat *Satellite) authoritive(req PubReq) bool { return false }
func (sat *Satellite) pub(m *hub.Msg, req PubReq) (*Update, error) {
	if len(req.Acts) == 0 {
		return nil, fmt.Errorf("no actions")
	}
	if req.Usr == "" {
		req.Usr = m.From.User()
	}
	req.Trans.Arrived = time.Now()
	// call the server and forward the reply if publish contains none-authoritative models
	if sat.auth == nil || !sat.auth(req.Acts) {
		if sat.isOnline() {
			return nil, fmt.Errorf("not connected")
		}
		nm := *m
		nm.Tok = sat.toks.Add(m)
		sat.cli.Chan() <- &nm
		return nil, nil // async
	}
	// otherwise publish authoritative models directly without revision
	// that means we will only reply to the sender and not the subsribers
	oldrev := sat.LocalRev()
	rev, evs, err := sat.PublishLocal(req.Trans)
	if err != nil {
		return nil, err
	}
	if sat.isOnline() {
		tr := req.Trans
		tr.Base = oldrev
		tr.Rev = rev
		tr.Acts = tr.Acts[:0]
		for _, ev := range evs {
			tr.Acts = append(tr.Acts, ev.Action)
		}
		sat.cli.Chan() <- &hub.Msg{Subj: m.Subj, Data: tr}
	}
	return showSubs(sat.Ctrl, m.From, oldrev, rev, evs), nil
}

func (sat *Satellite) handleRemote(m *hub.Msg) {
	switch m.Subj {
	case hub.Signoff:
		sat.status.Off = time.Now()
		sat.Input <- &hub.Msg{Subj: "evt.stat"}
	case hub.Signon:
		sat.status.Off = time.Time{}
		sat.status.On = time.Now()
		// send initial subscriptions
		sat.cli.Chan() <- &hub.Msg{Subj: "evt.sat", Data: SatReq{
			Rev: sat.Rev(), Trans: sat.Locals(), Tops: sat.tops,
		}}
		sat.Input <- &hub.Msg{Subj: "evt.stat"}
	case "evt.pub", "evt.sat", "evt.sub", "evt.update":
		upd, _ := m.Data.(*Update)
		if upd == nil {
			upd = &Update{}
			err := json.Unmarshal(m.Raw, upd)
			if err != nil {
				sat.Error("satellite unmarshal error", "err", err)
				return
			}
		}
		err := sat.Replicate(upd.Rev, upd.Evs)
		if err != nil {
			sat.Error("satellite replication error", "err", err)
			return
		}
		if m.Subj == "evt.pub" { // pass through to client
			err = sat.toks.Respond(m)
			if err != nil {
				sat.Error("satellite pub response error", "err", err)
			}
		}
	case "evt.unsub": // ignore
	default:
		sat.Error("satellite got unexpected remote message " + m.Subj)
	}
}
