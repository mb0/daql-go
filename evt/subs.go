package evt

import (
	"time"

	"xelf.org/daql/hub"
)

type Subscriber struct {
	hub.Conn
	Rev  time.Time
	Subs []string
	Mons []*Monitor
	Bufr []*Event
	Note bool

	monid int64 // last subscriber monitor id
}

type Monitor struct {
	Sub   *Subscriber
	ID    int64
	Watch []Watch
	Bufr  []Sig
}

func (s *Subscriber) Update(rev time.Time) *Update {
	if rev.After(s.Rev) {
		s.Rev = rev
	} else if !s.Note {
		return nil
	}
	res := &Update{Rev: rev, Evs: s.Bufr}
	s.Bufr = nil
	if s.Note {
		for _, m := range s.Mons {
			if len(m.Bufr) == 0 {
				continue
			}
			ws := sigsToWatch(m.Bufr)
			m.Bufr = nil
			res.Note = append(res.Note, Note{m.ID, ws})
		}
		s.Note = false
	}
	return res
}

type Subscribers struct {
	smap map[int64]*Subscriber
	tmap map[string][]*Subscriber
	mmap map[Sig][]*Monitor
}

func NewSubscribers() *Subscribers {
	return &Subscribers{
		smap: make(map[int64]*Subscriber),
		tmap: make(map[string][]*Subscriber),
		mmap: make(map[Sig][]*Monitor),
	}
}
func (subs *Subscribers) Get(id int64) *Subscriber { return subs.smap[id] }

// Show updates all matching subscribers with evs except for the sender itself and returns a sender
// subscription and an indicator whether a broadcast should be triggered.
// The returned sub is always usable to send a result update even if the sender was unknown.
func (subs *Subscribers) Show(from hub.Conn, evs []*Event) (sub *Subscriber, trig bool) {
	id := from.ID()
	for _, ev := range evs {
		for _, s := range subs.tmap[ev.Top] {
			if s.ID() == id {
				sub = s
			} else {
				trig = true
				s.Bufr = append(s.Bufr, ev)
			}
		}
		for _, m := range subs.mmap[ev.Sig] {
			trig = true
			m.Sub.Note = true
			m.Bufr = append(m.Bufr, ev.Sig)
		}
		if ev.Cmd == CmdCreate {
			for _, m := range subs.mmap[Sig{ev.Top, CmdCreate}] {
				trig = true
				m.Sub.Note = true
				m.Bufr = append(m.Bufr, ev.Sig)
			}
		}
	}
	// if the publishing connection is not itself an affected subscriber we create a dummy
	// subscription to simplify handling of the publish result update
	if sub == nil {
		sub = subs.smap[id]
		if sub == nil {
			sub = &Subscriber{Conn: from}
		}
	}
	return sub, trig
}

func (subs *Subscribers) Sub(c hub.Conn, rev time.Time, tops []string) (*Subscriber, []string) {
	id := c.ID()
	s := subs.smap[id]
	if len(tops) == 0 {
		return s, nil
	}
	if s == nil {
		s = &Subscriber{Conn: c}
		subs.smap[id] = s
	}
	res := make([]string, 0, len(tops))
	for _, t := range tops {
		idx := indexStr(s.Subs, t)
		if idx < 0 {
			s.Subs = append(s.Subs, t)
			res = append(res, t)
			subs.tmap[t] = append(subs.tmap[t], s)
		}
	}
	return s, res
}

func (subs *Subscribers) Unsub(c hub.Conn, tops []string) *Subscriber {
	id := c.ID()
	s, ok := subs.smap[id]
	if !ok {
		return nil
	}
	if len(tops) == 0 {
		tops = s.Subs
	}
	for _, t := range tops {
		idx := indexStr(s.Subs, t)
		if idx < 0 {
			continue
		}
		s.Subs = append(s.Subs[:idx], s.Subs[idx+1:]...)
		list := subs.tmap[t]
		for i, el := range list {
			if s == el {
				subs.tmap[t] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
	if len(s.Subs) == 0 && len(s.Mons) == 0 {
		delete(subs.smap, id)
		return s
	}
	// filter out buffered events for unsubscribed topics
	s.Bufr = filter(s.Bufr, tops)
	return s
}

func (subs *Subscribers) Mon(c hub.Conn, rev time.Time, ws []Watch) int64 {
	id := c.ID()
	s := subs.smap[id]
	if s == nil {
		s = &Subscriber{Conn: c}
		subs.smap[id] = s
	}
	s.monid++
	mon := &Monitor{Sub: s, ID: s.monid, Watch: ws}
	s.Mons = append(s.Mons, mon)
	for _, w := range ws {
		for _, k := range w.Keys {
			sig := Sig{w.Top, k}
			subs.mmap[sig] = append(subs.mmap[sig], mon)
		}
	}
	return mon.ID
}
func (subs *Subscribers) Unmon(c hub.Conn, mon int64) bool {
	id := c.ID()
	s := subs.smap[id]
	if s == nil {
		return false
	}
	var m *Monitor
	m, s.Mons = dropMon(s.Mons, mon)
	if m == nil {
		return false
	}
	for _, w := range m.Watch {
		for _, k := range w.Keys {
			sig := Sig{w.Top, k}
			_, mons := dropMon(subs.mmap[sig], m.ID)
			if len(mons) > 0 {
				subs.mmap[sig] = mons
			} else {
				delete(subs.mmap, sig)
			}
		}
	}
	if len(s.Mons) == 0 && len(s.Subs) == 0 {
		delete(subs.smap, id)
	}
	return true
}

// Bcast sends all buffered events up to revision rev out to subscribers.
func (subs *Subscribers) Bcast(from hub.Conn, rev time.Time) {
	for _, s := range subs.smap {
		res := s.Update(rev)
		if res != nil {
			s.Chan() <- &hub.Msg{From: from, Subj: "evt.update", Data: res}
		}
	}
}

func (subs *Subscribers) BcastMsg(msg *hub.Msg) {
	for _, s := range subs.smap {
		s.Chan() <- msg
	}
}

func indexStr(list []string, str string) int {
	for i, s := range list {
		if s == str {
			return i
		}
	}
	return -1
}

func dropMon(mons []*Monitor, id int64) (*Monitor, []*Monitor) {
	for i, m := range mons {
		if m.ID != id {
			continue
		}
		return m, append(mons[:i], mons[i+1:]...)
	}
	return nil, mons
}

func filter(evs []*Event, subs []string) []*Event {
	out := evs[:0] // reuse
	for _, ev := range evs {
		idx := indexStr(subs, ev.Top)
		if idx < 0 {
			out = append(out, ev)
		}
	}
	return out
}

func sigsToWatch(sigs []Sig) (res []Watch) {
	wm := make(map[string]*Watch)
	for _, sig := range sigs {
		w := wm[sig.Top]
		if w == nil {
			res = append(res, Watch{Top: sig.Top})
			w = &res[len(res)-1]
			wm[sig.Top] = w
		}
		w.Keys = append(w.Keys, sig.Key)
	}
	return res
}
