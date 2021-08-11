package wshub

import (
	"net/http"

	"github.com/gorilla/websocket"
	"xelf.org/daql/hub"
	"xelf.org/daql/log"
)

type Client struct {
	Config
	id   int64
	send chan *hub.Msg
}

func NewClient(conf Config) *Client {
	return &Client{Config: conf.Default(), id: hub.NextID(), send: make(chan *hub.Msg, 32)}
}

func (c *Client) ID() int64             { return c.id }
func (c *Client) Chan() chan<- *hub.Msg { return c.send }
func (c *Client) User() string          { return c.Config.User }

func (c *Client) Connect(r chan<- *hub.Msg) error {
	hdr, err := c.Token(c.URL)
	if err != nil {
		return err
	}
	wc, _, err := c.Dial(c.URL, hdr)
	if err != nil {
		c.ClearToken(c.URL)
		return err
	}
	cc := newConn(c.id, wc, c.send, c.Config.User)
	r <- &hub.Msg{From: cc, Subj: hub.Signon}
	go cc.writeAll(c.id, c.Log)
	err = cc.readAll(r)
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
	if c.Log == nil {
		c.Log = log.Root
	}
	if c.TokenProvider == nil {
		c.TokenProvider = (*nilProvider)(nil)
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
