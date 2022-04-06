package hub

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// lastID holds the last id returned from next id. It must only be accessed as atomic primitives.
var lastID = new(int64)

// NextID returns a new unused normal connection id.
func NextID() int64 { return int64(atomic.AddInt64(lastID, 1)) }

// Conn is a connection abstraction providing a ID, user field and a channel to send messages.
// Connections can represent one-off calls, connected clients of any kind, or the hub itself.
type Conn interface {
	// Ctx returns the connection context.
	Ctx() context.Context
	// ID is an internal connection identifier, the hub has id 0, transient connections have a
	// negative and normal connections positive ids.
	ID() int64
	// User is an external user identifier
	User() string
	// Chan returns an unchanging receiver channel. The hub sends a nil message to this
	// channel after a signoff message from this conn was received.
	Chan() chan<- *Msg
}

// ChanConn is a channel based connection used for simple in-process hub participants.
type ChanConn struct {
	ctx  context.Context
	id   int64
	user string
	ch   chan *Msg
}

// NewChanConn returns a new channel connection with the given id and channel.
func NewChanConn(ctx context.Context, id int64, user string, c chan *Msg) *ChanConn {
	return &ChanConn{ctx, id, user, c}
}

func (c *ChanConn) Ctx() context.Context { return c.ctx }
func (c *ChanConn) ID() int64            { return c.id }
func (c *ChanConn) User() string         { return c.user }
func (c *ChanConn) Chan() chan<- *Msg    { return c.ch }

// Req sends req to the hub from a newly created transient connection and returns the first response
// or an error if the timeout was reached.
func Req(hub chan<- *Msg, user string, req *Msg, timeout time.Duration) (*Msg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ch := make(chan *Msg, 1)
	req.From = NewChanConn(ctx, -1, user, ch)
	hub <- req
	select {
	case res := <-ch:
		if res == nil {
			return nil, fmt.Errorf("conn closed")
		}
		return res, nil
	case <-ctx.Done():
	}
	return nil, fmt.Errorf("timeout request %s#%s: %v", req.Subj, req.Tok, ctx.Err())
}

// Send sends a message to a connection that might have signed off and returns the success.
func Send(c Conn, m *Msg) bool {
	if c != nil {
		select {
		case c.Chan() <- m:
			return true
		default:
		}
	}
	return false
}
