// Package evt provides servers and tools for event sourcing.
package evt

const (
	CmdCreate = "+"
	CmdUpdate = "*"
	CmdDelete = "-"
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

type ByRev []*Event

func (s ByRev) Len() int           { return len(s) }
func (s ByRev) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByRev) Less(i, j int) bool { return s[j].Rev.After(s[i].Rev) }

type ByID []*Event

func (s ByID) Len() int           { return len(s) }
func (s ByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool { return s[i].ID < s[j].ID }
