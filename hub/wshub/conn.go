package wshub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/gorilla/websocket"
	"xelf.org/daql/hub"
	"xelf.org/daql/log"
	"xelf.org/xelf/bfr"
)

type conn struct {
	ctx  context.Context
	id   int64
	wc   *websocket.Conn
	user string
	send chan *hub.Msg
	tick <-chan time.Time
}

func newConn(ctx context.Context, id int64, wc *websocket.Conn, send chan *hub.Msg, user string) *conn {
	if send == nil {
		send = make(chan *hub.Msg, 32)
	}
	return &conn{ctx: ctx, id: id, wc: wc, send: send, user: user}
}

func (c *conn) Ctx() context.Context  { return c.ctx }
func (c *conn) ID() int64             { return c.id }
func (c *conn) Chan() chan<- *hub.Msg { return c.send }
func (c *conn) User() string          { return c.user }

func (c *conn) readAll(route chan<- *hub.Msg) error {
	for {
		if err := c.ctx.Err(); err != nil {
			return err
		}
		op, r, err := c.wc.NextReader()
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil // ignore error client disconnected
			}
			if cerr, ok := err.(*websocket.CloseError); ok && cerr.Code == 1001 {
				return nil // ignore error client disconnected
			}
			return fmt.Errorf("wshub client next reader: %w", err)
		}
		switch op {
		case websocket.BinaryMessage, websocket.TextMessage:
			// we treat text and binary messages the same and expect
			// a utf-8 header line with subj#tok\n
		case websocket.CloseMessage:
			return nil
		default:
			continue
		}
		raw, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("wshub read bytes failed: %w", err)
		}
		m, err := hub.Read(raw)
		if err != nil {
			return fmt.Errorf("wshub msg read failed: %w", err)
		}
		if privateSubj(m.Subj) {
			return fmt.Errorf("wshub client sent private message: %s", m.Subj)
		} else {
			m.From = c
			route <- m
		}
	}
}

func privateSubj(subj string) bool { return subj == "" || subj[0] == '_' }

func (c *conn) writeAll(id int64, log log.Logger, msgtimeout time.Duration) {
	defer c.wc.Close()
	for {
		select {
		case m := <-c.send:
			if m == nil {
				c.write(websocket.CloseMessage, []byte{}, 100*time.Millisecond)
				return
			}
			err := c.writeMsg(m, msgtimeout, log)
			if err != nil {
				return
			}
		case <-c.tick:
			err := c.write(websocket.PingMessage, []byte{}, time.Second)
			if err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *conn) write(kind int, data []byte, timeout time.Duration) error {
	c.wc.SetWriteDeadline(time.Now().Add(timeout))
	return c.wc.WriteMessage(kind, data)
}

func (c *conn) writeMsg(msg *hub.Msg, timeout time.Duration, log log.Logger) error {
	b := bfr.Get()
	defer bfr.Put(b)
	kind, err := writeMsgTo(&bfr.P{Writer: b, JSON: true}, msg)
	if err != nil {
		return err
	}
	return c.write(kind, b.Bytes(), timeout)
}

type msgKind int

var AsBinary msgKind = websocket.BinaryMessage

func writeMsgTo(b *bfr.P, m *hub.Msg) (kind int, err error) {
	b.Fmt(m.Subj)
	if len(m.Tok) != 0 {
		b.Byte('#')
		b.Fmt(m.Tok)
	}
	err = b.Byte('\n')
	if m.Data == AsBinary {
		_, err = b.Write(m.Raw)
		return websocket.BinaryMessage, err
	} else if len(m.Raw) != 0 {
		_, err = b.Write(m.Raw)
	} else if m.Data != nil {
		if w, ok := m.Data.(bfr.Printer); ok {
			err = w.Print(b)
		} else {
			err = json.NewEncoder(b).Encode(m.Data)
		}
	}
	return websocket.TextMessage, err
}
