// Package evt provides servers and tools for event sourcing.
package evt

import (
	"fmt"

	"xelf.org/xelf/lit"
)

const (
	CmdNew = "new"
	CmdMod = "mod"
	CmdDel = "del"
)

func Collect(evs []*Event, s Sig) (res []*Event) {
	for _, ev := range evs {
		if ev.Sig == s {
			res = append(res, ev)
		}
	}
	return res
}

func CollectAll(evs []*Event) map[Sig][]*Event {
	res := make(map[Sig][]*Event)
	for _, ev := range evs {
		res[ev.Sig] = append(res[ev.Sig], ev)
	}
	return res
}

func Merge(a, b Action) (_ Action, err error) {
	if a.Sig != b.Sig {
		return a, fmt.Errorf("event signature mismatch %v != %v", a.Sig, b.Sig)
	}
	switch a.Cmd {
	case CmdDel:
		switch b.Cmd {
		case CmdNew:
			// TODO zero optional arg fields
			return Action{Sig: a.Sig, Cmd: CmdMod, Arg: b.Arg}, nil
		case CmdMod:
			return a, fmt.Errorf("mod after del action for %v", a.Sig)
		case CmdDel:
			return a, fmt.Errorf("double del action for %v", a.Sig)
		}
	case CmdNew, CmdMod:
		switch b.Cmd {
		case CmdNew:
			return a, fmt.Errorf("new action for existing %v", a.Sig)
		case CmdMod:
			if a.Cmd == CmdNew {
				return a, lit.Apply(a.Arg, lit.Delta(b.Arg.Keyed))
			}
			return a, MergeDeltas(lit.Delta(a.Arg.Keyed), lit.Delta(b.Arg.Keyed))
		case CmdDel:
			return b, nil
		}
	default:
		return a, fmt.Errorf("unresolved action %s", a.Cmd)
	}
	return a, fmt.Errorf("unresolved action %s", b.Cmd)
}

func MergeDeltas(a, b lit.Delta) error {
	for k, v := range b {
		// TODO check for common prefix, but we use flat updates for now
		a[k] = v
	}
	return nil
}

type ByRev []*Event

func (s ByRev) Len() int           { return len(s) }
func (s ByRev) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByRev) Less(i, j int) bool { return s[j].Rev.After(s[i].Rev) }

type ByID []*Event

func (s ByID) Len() int           { return len(s) }
func (s ByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool { return s[i].ID < s[j].ID }
