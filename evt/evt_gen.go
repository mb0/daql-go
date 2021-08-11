// generated code

package evt

import (
	"encoding/json"
	"time"
	"xelf.org/daql/hub"
	"xelf.org/daql/mig"
	"xelf.org/xelf/lit"
)

// Audit holds detailed information for a published revision.
type Audit struct {
	Rev     time.Time          `json:"rev"`
	Created time.Time          `json:"created,omitempty"`
	Arrived time.Time          `json:"arrived,omitempty"`
	User    string             `json:"user,omitempty"`
	Extra   map[string]lit.Val `json:"extra,omitempty"`
}

// Sig is the event signature.
type Sig struct {
	Top string `json:"top"`
	Key string `json:"key"`
}

// Action is an unpublished event represented by a command string and argument map.
// It usually is a data operation on a record identified by a topic and primary key.
type Action struct {
	Sig
	Cmd string             `json:"cmd"`
	Arg map[string]lit.Val `json:"arg,omitempty"`
}

// Event is an action published to a ledger with revision and unique id.
type Event struct {
	ID  int64     `json:"id"`
	Rev time.Time `json:"rev"`
	Action
}

// Trans is an request to publish a list of actions for a base revision.
type Trans struct {
	ID   int64     `json:"id"`
	Base time.Time `json:"base"`
	Audit
	Acts []Action `json:"acts"`
}

// Watch is topic name and list of keys to monitor.
type Watch struct {
	Top  string   `json:"top"`
	Keys []string `json:"keys"`
}

type Note struct {
	Mon   int64   `json:"mon"`
	Watch []Watch `json:"watch"`
}

// Update holds a list of events and notes
type Update struct {
	Rev  time.Time `json:"rev"`
	Evs  []*Event  `json:"evs,omitempty"`
	Note []Note    `json:"note,omitempty"`
}

// Status holds the current ledger revision migration information.
type Status struct {
	Rev time.Time     `json:"rev"`
	Mig mig.Migration `json:"mig"`
	On  time.Time     `json:"on,omitempty"`
	Off time.Time     `json:"off,omitempty"`
}

type StatRes struct {
	Res Status `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type StatFunc func(*hub.Msg) (Status, error)

func (f StatFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	res, err := f(m)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type PubReq struct {
	Trans
}

type PubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type PubFunc func(*hub.Msg, PubReq) (*Update, error)

func (f PubFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req PubReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type SubReq struct {
	Rev  time.Time `json:"rev"`
	Tops []string  `json:"tops"`
}

type SubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type SubFunc func(*hub.Msg, SubReq) (*Update, error)

func (f SubFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req SubReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type SatReq struct {
	Rev   time.Time `json:"rev"`
	Trans []Trans   `json:"trans"`
	Tops  []string  `json:"tops"`
}

type SatRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type SatFunc func(*hub.Msg, SatReq) (*Update, error)

func (f SatFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req SatReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type UnsubReq struct {
	Tops []string `json:"tops"`
}

type UnsubRes struct {
	Res bool   `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type UnsubFunc func(*hub.Msg, UnsubReq) (bool, error)

func (f UnsubFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req UnsubReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type MonReq struct {
	Rev   time.Time `json:"rev"`
	Watch []Watch   `json:"watch"`
}

type MonRes struct {
	Res int64  `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type MonFunc func(*hub.Msg, MonReq) (int64, error)

func (f MonFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req MonReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}

type UnmonReq struct {
	Mon int64 `json:"mon"`
}

type UnmonRes struct {
	Res bool   `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type UnmonFunc func(*hub.Msg, UnmonReq) (bool, error)

func (f UnmonFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req UnmonReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}
