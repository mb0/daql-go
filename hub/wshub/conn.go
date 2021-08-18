package wshub

import (
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
	id   int64
	wc   *websocket.Conn
	user string
	send chan *hub.Msg
	tick <-chan time.Time
}

func newConn(id int64, wc *websocket.Conn, send chan *hub.Msg, user string) *conn {
	if send == nil {
		send = make(chan *hub.Msg, 32)
	}
	return &conn{id: id, wc: wc, send: send, user: user}
}

func (c *conn) ID() int64             { return c.id }
func (c *conn) Chan() chan<- *hub.Msg { return c.send }
func (c *conn) User() string          { return c.user }

func (c *conn) readAll(route chan<- *hub.Msg) error {
	for {
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
		if op == websocket.BinaryMessage {
			return fmt.Errorf("wshub client unexpected binary message: %w", err)
		}
		if op != websocket.TextMessage {
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

func privateSubj(subj string) bool {
	switch subj {
	case "", "+", "-":
		return true
	}
	return subj[0] == '_'
}

func (c *conn) writeAll(id int64, log log.Logger) {
	defer c.wc.Close()
	for {
		select {
		case m := <-c.send:
			if m == nil {
				c.write(websocket.CloseMessage, []byte{}, time.Second)
				return
			}
			err := c.writeMsg(m, 20*time.Second, log)
			if err != nil {
				return
			}
		case <-c.tick:
			err := c.write(websocket.PingMessage, []byte{}, 5*time.Second)
			if err != nil {
				return
			}
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
	if err := writeMsgTo(&bfr.P{Writer: b, JSON: true}, msg); err != nil {
		log.Error("write msg", "err", err)
		return err
	}
	return c.write(websocket.TextMessage, b.Bytes(), timeout)
}

func writeMsgTo(b *bfr.P, m *hub.Msg) error {
	b.Fmt(m.Subj)
	if len(m.Tok) != 0 {
		b.Byte('#')
		b.Fmt(m.Tok)
	}
	b.Byte('\n')
	if len(m.Raw) != 0 {
		b.Write(m.Raw)
	} else if m.Data != nil {
		if w, ok := m.Data.(bfr.Printer); ok {
			return w.Print(b)
		}
		return json.NewEncoder(b).Encode(m.Data)
	}
	return b.Err
}
