package ses

import (
	"fmt"
	"net/http"

	"xelf.org/daql/log"
)

type Manager struct {
	Config   []Config
	Store    Store
	Log      log.Logger
	Decorate func(*http.Request, *Session) *http.Request
}

func NewManager(store Store, conf ...Config) *Manager {
	return &Manager{Config: conf, Store: store, Decorate: Decorate}
}

func (m *Manager) Clear(w http.ResponseWriter, r *http.Request) {
	for _, c := range m.Config {
		if tw, ok := c.TokenReader.(TokenWriter); ok {
			tw.WriteToken(w, "")
			if td, err := m.readAndDecode(c, r); err == nil {
				m.Store.Delete(td)
			}
		}
	}
}

func (m *Manager) Read(w http.ResponseWriter, r *http.Request) (_ *Session, err error) {
	var td string
	var tw TokenWriter
	for _, c := range m.Config {
		td, err = m.readAndDecode(c, r)
		if err != nil {
			continue
		}
		tw, _ = c.TokenReader.(TokenWriter)
		break
	}
	if td == "" {
		return nil, errNoToken
	}
	data, err := m.Store.Get(td)
	if err != nil {
		if tw != nil && w != nil { // clear outdated cookie
			tw.WriteToken(w, "")
		}
		return nil, err
	}
	return &Session{Data: data, IsCookie: tw != nil}, nil
}

func (m *Manager) Save(w http.ResponseWriter, s *Session) error {
	if s == nil {
		return nil
	}
	if s.IsNew {
		if s.IsCookie {
			err := m.EncodeAndWrite(w, s)
			if err != nil {
				return err
			}
		}
		if m.Log != nil {
			m.Log.Debug("session created", "ses", s.ID())
		}
	}
	return m.Store.Save(s.Data, s.IsNew)
}

func (m *Manager) ReadOrCreate(w http.ResponseWriter, r *http.Request, cookie bool) *http.Request {
	// read session from cookie or header
	s, err := m.Read(w, r)
	if err != nil {
		// session read failed
		data, err := m.Store.New()
		if err != nil {
			log.Error("create session failed", "err", err)
		}
		s = &Session{Data: data, IsNew: true, IsCookie: cookie}
	}
	if m.Decorate != nil {
		r = m.Decorate(r, s)
	}
	return r
}

func (m *Manager) EncodeAndWrite(w http.ResponseWriter, s *Session) (err error) {
	for _, c := range m.Config {
		err = m.encodeAndWrite(c, w, s)
		if err == nil {
			return nil
		}
	}
	return err
}

func (m *Manager) readAndDecode(c Config, r *http.Request) (string, error) {
	tok := c.ReadToken(r)
	if tok != "" {
		for _, cc := range c.Codec {
			data, err := cc.DecodeToken(tok)
			if err != nil || data == "" {
				continue
			}
			return data, nil
		}
	}
	return "", errNoToken
}

func (m *Manager) encodeAndWrite(c Config, w http.ResponseWriter, s *Session) (err error) {
	tw, ok := c.TokenReader.(TokenWriter)
	if !ok {
		return fmt.Errorf("no token writer")
	}
	var tok string
	for _, cc := range c.Codec {
		data := s.Tok()
		tok, err = cc.EncodeToken(data)
		if err != nil {
			l := m.Log
			if l == nil {
				l = log.Root
			}
			l.Error(fmt.Sprintf("ses codec %T failed to encode %q", cc, data))
			continue
		}
	}
	if tok == "" {
		return errNoToken
	}
	tw.WriteToken(w, tok)
	return nil
}
