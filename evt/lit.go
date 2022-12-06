package evt

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"xelf.org/daql/dom"
	"xelf.org/daql/qry"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// MemLedger implements an in-memory ledger.
type MemLedger struct {
	Reg  lit.Regs
	Bend *qry.MemBackend
	evs  []*Event
}

// NewMemLedger returns a new ledger for testing, that is backed by the memory query backend b. This
// ledger provides no persistence and expands only minimal effort to roll back after a failed event
// publish. It should only be used for testing or if these constraints are well understood.
// The ledger assumes sole and full control over b and converts event data to an event slice
// and swaps previous event values with proxies.
func NewMemLedger(reg *lit.Regs, b *qry.MemBackend) (*MemLedger, error) {
	reg = lit.DefaultRegs(reg)
	m := b.Project.Model("evt.event")
	if m == nil {
		return nil, fmt.Errorf("mem ledger: backend has no event model")
	}
	list := b.Data["evt.Model"]
	var evs []*Event
	if list != nil && len(list.Vals) > 0 {
		evs = make([]*Event, 0, len(list.Vals))
		for i, v := range list.Vals {
			ev := new(Event)
			prx, err := lit.Proxy(reg, ev)
			if err != nil {
				return nil, err
			}
			err = prx.Assign(v)
			if err != nil {
				return nil, err
			}
			evs = append(evs, ev)
			list.Vals[i] = prx
		}
		sort.Stable(evtVals{evs, list.Vals})
	}
	return &MemLedger{Reg: *reg, Bend: b, evs: evs}, nil
}
func (l *MemLedger) Rev() time.Time {
	if len(l.evs) > 0 {
		return l.evs[len(l.evs)-1].Rev
	}
	return time.Time{}
}
func (l *MemLedger) Project() *dom.Project { return l.Bend.Project }

func (l *MemLedger) Events(rev time.Time, tops ...string) (res []*Event, _ error) {
	var m map[string]struct{}
	if len(tops) > 0 {
		m = make(map[string]struct{}, len(tops))
		for _, t := range tops {
			m[t] = struct{}{}
		}
	}
	for _, ev := range l.evs {
		if !ev.Rev.After(rev) {
			continue
		}
		if m == nil {
			res = append(res, ev)
		} else if _, ok := m[ev.Top]; ok {
			res = append(res, ev)
		}
	}
	return res, nil
}

// Publish publishes transaction t the ledger it attempts to roll back failed transactions.
// Failed reverts may panic. Use only for testing.
func (l *MemLedger) Publish(t Trans) (time.Time, []*Event, error) {
	rev := l.Rev()
	if t.Base.IsZero() {
		t.Base = rev
	} else if t.Base.After(rev) {
		return rev, nil, fmt.Errorf("publish: future base revision")
	}
	if len(t.Acts) == 0 {
		return rev, nil, fmt.Errorf("publish: no actions")
	}
	now := time.Now()
	if t.Arrived.IsZero() {
		t.Arrived = now
	}
	if t.Created.IsZero() {
		t.Created = now
	}
	evs := make([]*Event, 0, len(t.Acts))
	nrev := NextRev(rev, now)
	check := rev.After(t.Base)
	var keys []string
	for _, act := range t.Acts {
		if check && act.Cmd != CmdNew {
			// collect the keys to look for conflicts
			keys = append(keys, act.Key)
		}
		evs = append(evs, &Event{Rev: nrev, Action: act})
	}
	if check && len(keys) > 0 {
		// TODO check for conflict
	}
	// roll back on failed event application
	// we cannot easily defer modification of the backend until we know that we can apply
	// all events (e.g. deletions would alter later indexes or influence a later duplicate).
	// therefor we need to reverse modification for already applied events.
	var reverts []func() error
	for _, ev := range evs {
		revert, err := l.applyEvent(ev)
		if err != nil {
			for i := len(reverts) - 1; i >= 0; i++ {
				er := reverts[i]()
				if er != nil {
					panic(fmt.Errorf("revert err: %v\nafter apply: %v", er, err))
				}
			}
			return rev, nil, err
		}
		reverts = append(reverts, revert)
	}
	// insert event or audit error is a system error that should not depend on user input
	err := l.insertEvents(evs)
	if err != nil {
		return rev, nil, err
	}
	// TODO insert audit
	// insertAudit(c, rev, t.Detail)
	return nrev, evs, nil
}

func (l *MemLedger) insertEvents(evs []*Event) error {
	var last int64
	if len(l.evs) > 0 {
		last = l.evs[len(l.evs)-1].ID
	}
	list := l.Bend.Data["evt.event"]
	if list == nil {
		list = &lit.List{Typ: typ.List}
		l.Bend.Data["evt.event"] = list
	}
	for _, ev := range evs {
		last++
		ev.ID = last
		prx, err := lit.Proxy(l.Reg, ev)
		if err != nil {
			return err
		}
		l.evs = append(l.evs, ev)
		list.Vals = append(list.Vals, prx)
	}
	return nil
}
func (l *MemLedger) applyEvent(ev *Event) (func() error, error) {
	m := l.Bend.Project.Model(ev.Top)
	if m == nil {
		return nil, fmt.Errorf("no model found for topic %s", ev.Top)
	}
	pk, pt, err := primaryKey(m)
	if err != nil {
		return nil, err
	}
	d := l.Bend.Data[ev.Top]
	switch ev.Cmd {
	case CmdDel:
		// find by ev.Key
		idx, err := indexKey(d, pk, ev.Key)
		if err != nil {
			return nil, fmt.Errorf("apply del %s: %w", ev.Top, err)
		}
		if idx < 0 {
			return nil, fmt.Errorf("apply del %s, %s: no value found ", ev.Top, ev.Key)
		}
		// delete from vals
		del := d.Vals[idx]
		d.Vals = append(d.Vals[:idx], d.Vals[idx+1:]...)
		return func() error {
			d.Vals = append(d.Vals[:idx+1], d.Vals[idx:]...)
			d.Vals[idx] = del
			return nil
		}, nil
	case CmdNew:
		if d == nil {
			d = lit.NewList(m.Type())
			l.Bend.Data[ev.Top] = d
		}
		val := l.Reg.Zero(m.Type())
		mut := val.(lit.Keyr)
		kv, err := keyVal(pt, ev.Key)
		if err != nil {
			return nil, fmt.Errorf("apply new %s: %w", ev.Top, err)
		}
		err = mut.SetKey(pk, kv)
		if err != nil {
			return nil, fmt.Errorf("apply new %s %s: %w", ev.Top, pk, err)
		}
		_, err = lit.Apply(mut, lit.Delta(ev.Arg.Keyed))
		if err != nil {
			return nil, fmt.Errorf("apply new %s arg: %w", ev.Top, err)
		}
		if hasRev(m) {
			err = mut.SetKey("rev", lit.Time(ev.Rev))
			if err != nil {
				return nil, fmt.Errorf("apply new %s rev: %w", ev.Top, err)
			}
		}
		idx := len(d.Vals)
		d.Vals = append(d.Vals, mut)
		return func() error {
			d.Vals = d.Vals[:idx]
			return nil
		}, nil
	case CmdMod:
		// find by ev.Key
		idx, err := indexKey(d, pk, ev.Key)
		if err != nil {
			return nil, fmt.Errorf("apply mod %s: %w", ev.Top, err)
		}
		if idx < 0 {
			return nil, fmt.Errorf("apply mod %s, %s: no value found ", ev.Top, ev.Key)
		}
		mut := d.Vals[idx].(lit.Keyr)
		org := mut.String()
		// mod arg
		_, err = lit.Apply(mut, lit.Delta(ev.Arg.Keyed))
		if err != nil {
			return nil, fmt.Errorf("apply mod %s arg: %w", ev.Top, err)
		}
		if hasRev(m) {
			err = mut.SetKey("rev", lit.Time(ev.Rev))
			if err != nil {
				return nil, fmt.Errorf("apply mod %s rev: %w", ev.Top, err)
			}
		}
		// write back
		d.Vals[idx] = mut
		return func() error {
			return lit.ParseInto(org, mut)
		}, nil
	}
	return nil, fmt.Errorf("unknown command %s", ev.Cmd)
}
func primaryKey(m *dom.Model) (string, typ.Type, error) {
	for _, f := range m.Elems {
		if f.Bits&dom.BitPK != 0 {
			return cor.Keyed(f.Name), f.Type, nil
		}
	}
	return "", typ.Void, fmt.Errorf("no pk field for model %s", m.Qualified())
}
func hasRev(m *dom.Model) bool {
	for _, f := range m.Elems {
		if f.Name == "Rev" && f.Type.Kind&knd.Time != 0 {
			return true
		}
	}
	return false
}
func keyVal(t typ.Type, key string) (lit.Val, error) {
	if t.Kind&knd.Char != 0 {
		return lit.Str(key), nil
	}
	if t.Kind&knd.Int != 0 {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, err
		}
		return lit.Int(id), nil
	}
	return nil, fmt.Errorf("unexpected key type %s", t)
}

func indexKey(list *lit.List, pk, key string) (int, error) {
	if list == nil {
		return -1, fmt.Errorf("no data found")
	}
	for i, v := range list.Vals {
		id, err := v.(lit.Keyr).Key(pk)
		if err != nil {
			return -1, err
		}
		if id.String() == key {
			return i, nil
		}
	}
	return -1, nil
}

type evtVals struct {
	evt []*Event
	val []lit.Val
}

func (s evtVals) Len() int { return len(s.evt) }
func (s evtVals) Swap(i, j int) {
	s.evt[i], s.evt[j] = s.evt[j], s.evt[i]
	s.val[i], s.val[j] = s.val[j], s.val[i]
}
func (s evtVals) Less(i, j int) bool {
	a, b := s.evt[i], s.evt[j]
	if a.ID == b.ID {
		return !a.Rev.After(b.Rev)
	}
	return a.ID < b.ID
}
