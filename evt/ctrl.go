package evt

import (
	"fmt"
	"sort"
	"time"

	"xelf.org/daql/hub"
	"xelf.org/daql/log"
)

// Ctrl manages subscription updates common to both the event server and satellite.
type Ctrl struct {
	Ledger
	Input chan *hub.Msg
	Subs  *Subscribers
	log.Logger

	timer *time.Timer
	btrig time.Time
	bcast time.Time
}

// NewCtrl returns a new controller for ledger l.
func NewCtrl(l Ledger) *Ctrl {
	return &Ctrl{Ledger: l,
		Input:  make(chan *hub.Msg, 64),
		Subs:   NewSubscribers(),
		Logger: log.Root,
	}
}

func (ctr *Ctrl) Services() hub.Services {
	return hub.Services{
		"evt.sub":   SubFunc(ctr.sub),
		"evt.unsub": UnsubFunc(ctr.unsub),
		"evt.mon":   MonFunc(ctr.mon),
		"evt.unmon": UnmonFunc(ctr.unmon),
	}
}

func (ctr *Ctrl) Handle(m *hub.Msg) {
	switch m.Subj {
	case "_btrig":
		ctr.Btrig()
	case "_bcast":
		ctr.Bcast(m.From, ctr.Rev())
	case "_stop":
		ctr.Stop()
	case hub.Signoff:
		ctr.Subs.Unsub(m.From, nil)
	default:
		ctr.Error("evt server message", "subj", m.Subj)
	}
}

func (ctr *Ctrl) sub(m *hub.Msg, req SubReq) (*Update, error) {
	s, tops := ctr.Subs.Sub(m.From, req.Rev, req.Tops)
	if len(tops) == 0 {
		return nil, fmt.Errorf("no new subscriptions")
	}
	evs, err := ctr.Events(req.Rev, tops...)
	if err != nil {
		return nil, err
	}
	if len(s.Bufr) == 0 {
		s.Bufr = evs
	} else {
		s.Bufr = append(evs, s.Bufr...)
		sort.Sort(ByRev(s.Bufr))
	}
	return s.Update(ctr.Rev()), nil
}
func (ctr *Ctrl) unsub(m *hub.Msg, req UnsubReq) (bool, error) {
	return ctr.Subs.Unsub(m.From, req.Tops) != nil, nil
}
func (ctr *Ctrl) mon(m *hub.Msg, req MonReq) (int64, error) {
	return ctr.Subs.Mon(m.From, req.Rev, req.Watch), nil
}
func (ctr *Ctrl) unmon(m *hub.Msg, req UnmonReq) (bool, error) {
	return ctr.Subs.Unmon(m.From, req.Mon), nil
}

// Btrig throttles a trigger to send a _bcast messages at least 200ms apart.
func (ctr *Ctrl) Btrig() {
	// ignore if timer is still going to be called
	now := time.Now()
	if now.Sub(ctr.btrig) < 200*time.Millisecond {
		return
	}
	// otherwise send a delayed bcast message
	ctr.btrig = now
	ctr.timer = time.AfterFunc(200*time.Millisecond, func() {
		ctr.Input <- &hub.Msg{Subj: "_bcast"}
	})
}

// Bcast sends all buffered events up to revision rev out to subscribers.
func (ctr *Ctrl) Bcast(from hub.Conn, rev time.Time) {
	if !rev.After(ctr.bcast) {
		return
	}
	ctr.bcast = rev
	ctr.Subs.Bcast(from, rev)
}

func (ctr *Ctrl) Stop() {
	if ctr.timer != nil {
		ctr.timer.Stop()
	}
}
