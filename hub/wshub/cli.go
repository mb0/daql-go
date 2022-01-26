package wshub

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"xelf.org/daql/hub"
	"xelf.org/daql/log"
)

// Client is connection to a hub served over websockets.
type Client struct {
	Config
	ctx  context.Context
	id   int64
	send chan *hub.Msg
}

// WSURL returns url with a http or https prefix replaced as ws or wss respectively.
func WSURL(url string) string {
	if strings.HasPrefix(url, "http") {
		url = "ws" + url[4:]
	}
	return url
}

// NewClient returns a new client with the given configuration.
func NewClient(ctx context.Context, conf Config) *Client {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Client{Config: conf.Default(), ctx: ctx, id: hub.NextID(), send: make(chan *hub.Msg, 32)}
}

func (c *Client) Ctx() context.Context  { return c.ctx }
func (c *Client) ID() int64             { return c.id }
func (c *Client) Chan() chan<- *hub.Msg { return c.send }
func (c *Client) User() string          { return c.Config.User }

// Start connects the client and continues sending incoming messages to r or returns an error.
func (c *Client) Start(r chan<- *hub.Msg) error {
	wc, err := c.connect()
	if err != nil {
		return err
	}
	go c.run(wc, r)
	return nil
}

// Run connects the client and blocks while sending incoming messages to r and returns an error.
func (c *Client) Run(r chan<- *hub.Msg) error {
	wc, err := c.connect()
	if err != nil {
		return err
	}
	return c.run(wc, r)
}

// Backoff returns a duration to sleep before reconnecting or an error.
type Backoff func(retry int, err error) (time.Duration, error)

// RunWithBackoff blocks while running and reconnecting using a backoff function.
func (c *Client) RunWithBackoff(r chan<- *hub.Msg, bof Backoff) error {
	for nerr := 0; ; {
		wc, err := c.connect()
		if err != nil {
			nerr += 1
		} else {
			nerr = 0
			err = c.run(wc, r)
		}
		if err := c.ctx.Err(); err != nil {
			return err
		}
		sleep, err := bof(nerr, err)
		if err != nil {
			return err
		}
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

func (c *Client) connect() (*websocket.Conn, error) {
	hdr, err := c.Token(c.URL)
	if err != nil {
		return nil, err
	}
	wc, _, err := c.DialContext(c.ctx, c.URL, hdr)
	if err != nil {
		c.ClearToken(c.URL)
		return nil, err
	}
	c.Log.Debug("wshub client connected", "url", c.URL)
	return wc, nil
}

func (c *Client) run(wc *websocket.Conn, r chan<- *hub.Msg) error {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	cc := newConn(ctx, c.id, wc, c.send, c.Config.User)
	r <- &hub.Msg{From: cc, Subj: hub.Signon}
	go cc.writeAll(c.id, c.Log, 2*time.Second)
	err := cc.readAll(r)
	c.send <- nil
	r <- &hub.Msg{From: cc, Subj: hub.Signoff}
	return err
}

type Config struct {
	URL  string
	User string
	*websocket.Dialer
	TokenProvider
	Log log.Logger
}

func (c Config) Default() Config {
	if c.Dialer == nil {
		c.Dialer = websocket.DefaultDialer
	}
	if c.TokenProvider == nil {
		c.TokenProvider = (*nilProvider)(nil)
	}
	if c.Log == nil {
		c.Log = log.Root
	}
	return c
}

type TokenProvider interface {
	Token(url string) (http.Header, error)
	ClearToken(url string) error
}

type nilProvider struct{}

func (*nilProvider) Token(string) (http.Header, error) { return nil, nil }
func (*nilProvider) ClearToken(string) error           { return nil }
