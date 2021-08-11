package evt

import (
	"time"

	"xelf.org/daql/dom"
)

// NextRev returns a rev truncated to ms or if rev is not after last the next possible revision one
// millisecond after the last.
func NextRev(last, rev time.Time) time.Time {
	rev = rev.Truncate(time.Millisecond)
	if rev.After(last) {
		return rev
	}
	return last.Add(time.Millisecond)
}

// Ledger abstracts over the event storage. It allows to access the latest revision and query
// events. Ledger implementations are usually not thread-safe unless explicitly documented.
type Ledger interface {
	// Rev returns the latest event revision or the zero time.
	Rev() time.Time
	Project() *dom.Project
	// Events returns all events for the given topics since rev.
	// This methods is primarily used by the event central to manage subscribed events.
	// The qry package can be used for more complex event queries.
	Events(rev time.Time, tops ...string) ([]*Event, error)
}

// Publisher is a ledger that can publish transactions.
type Publisher interface {
	Ledger
	Publish(Trans) (time.Time, []*Event, error)
}

// Replicator is a ledger that can replicate events.
type Replicator interface {
	Ledger
	Replicate([]*Event) error
}

// LocalPublisher is a replicator that can publish events locally.
type LocalPublisher interface {
	Replicator
	PublishLocal(Trans) ([]*Event, error)
	Locals() []Trans
}
