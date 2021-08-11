// Package hub provides a transport agnostic connection hub.
package hub

import (
	"regexp"
	"strings"
)

const (
	Signon  = "_signon"
	Signoff = "_signoff"
)

// Hub is the central message server that manages connection. Hub itself represents conn id 0.
// Connection creators are also responsible for sending the signon message and validating messages.
// One-off connections used for a simple request-response round trips can be used without signon and
// must use id -1. These connections can only be responded to directly and must not be stored.
type Hub struct {
	conns map[int64]Conn
	msgs  chan *Msg
}

// NewHub creates and returns a new hub.
func NewHub() *Hub {
	return &Hub{conns: make(map[int64]Conn, 64), msgs: make(chan *Msg, 128)}
}

func (h *Hub) ID() int64         { return 0 }
func (h *Hub) Chan() chan<- *Msg { return h.msgs }
func (h *Hub) User() string      { return "hub" }

// Run starts routing received messages with the given router. It is usually run as go routine.
func (h *Hub) Run(r Router) {
	for m := range h.msgs {
		if m == nil {
			break
		}
		if m.Subj == Signon {
			h.conns[m.From.ID()] = m.From
		}
		r.Route(m)
		if m.Subj == Signoff {
			delete(h.conns, m.From.ID())
			m.From.Chan() <- nil
		}
	}
}

// Router routes a received message to connection.
type Router interface{ Route(*Msg) }

// Routers is a slice of routers, all of them are called with incoming messages.
type Routers []Router

func (rs Routers) Route(m *Msg) {
	for _, r := range rs {
		r.Route(m)
	}
}

// RouterFunc implements Router for simple route functions.
type RouterFunc func(*Msg)

func (r RouterFunc) Route(m *Msg) { r(m) }

// MatchFilter only routes messages, that match one of a list of subjects.
type MatchFilter struct {
	Router
	Match []string
}

// NewMatchFilter returns a new router that matches message subject and calls r.
func NewMatchFilter(r Router, match ...string) RouterFunc {
	return func(m *Msg) {
		for _, s := range match {
			if m.Subj == s {
				r.Route(m)
				return
			}
		}
	}
}

// NewMatchFilter returns a new router that matches message subject prefixes and calls r.
func NewPrefixFilter(r Router, match ...string) RouterFunc {
	return func(m *Msg) {
		for _, s := range match {
			if strings.HasPrefix(m.Subj, s) {
				r.Route(m)
				return
			}
		}
	}
}

// NewRegexpFilter returns a new router that matches message subjects with regexp and calls r.
func NewRegexpFilter(r Router, pat *regexp.Regexp) RouterFunc {
	return func(m *Msg) {
		if pat.MatchString(m.Subj) {
			r.Route(m)
		}
	}
}
