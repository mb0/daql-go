package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Msg is the central structure passed between connections. The optional body is represented by raw
// bytes or typed data. If raw bytes are required but not set, non nil data is encoded as JSON.
type Msg struct {
	// From is the origin connection of this message or nil for server internal messages.
	From Conn
	// Subj is the required message header used for routing and determining the data type.
	Subj string
	// Tok is a client token that is used in replies, so they can be matched to a request.
	Tok string
	// Raw is the message body as bytes usually encoded as JSON.
	Raw []byte
	// Data is the typed body data and can be used to avoid serialization of internal messages.
	Data interface{}
}

// Parse parses str and returns a message or an error.
func Parse(str string) (*Msg, error) { return Read([]byte(str)) }

// Read parses input bytes and returns a message or an error.
// The byte slice is then owned by the message and cannot be reused.
func Read(input []byte) (*Msg, error) {
	var subj, tok, raw []byte
	subj = input
	idx := bytes.IndexByte(subj, '\n')
	if idx >= 0 {
		subj, raw = subj[:idx], append(raw, subj[idx+1:]...)
	}
	idx = bytes.IndexByte(subj, '#')
	if idx >= 0 {
		subj, tok = subj[:idx], subj[idx+1:]
	}
	if len(subj) == 0 {
		return nil, fmt.Errorf("message without subject")
	}
	return &Msg{Subj: string(subj), Tok: string(tok), Raw: raw}, nil
}

// String returns the default string format of this message.
func (m *Msg) String() string {
	r := m.Raw
	if len(r) == 0 && m.Data != nil {
		r, _ = json.Marshal(m.Data)
	}
	return fmt.Sprintf("%s#%s\n%s", m.Subj, m.Tok, r)
}

func (m *Msg) Unmarshal(v interface{}) error {
	if m.Raw == nil {
		return fmt.Errorf("no data for msg %s", m.Subj)
	}
	err := json.Unmarshal(m.Raw, v)
	if err != nil {
		return err
	}
	m.Data = v
	return nil
}

func (m *Msg) Reply(data interface{}) *Msg {
	raw, err := json.Marshal(data)
	if err != nil {
		return m.ReplyErr(err)
	}
	return &Msg{Subj: m.Subj, Tok: m.Tok, Raw: raw}
}

func (m *Msg) ReplyRes(res interface{}) *Msg { return m.Reply(resData{Res: res}) }
func (m *Msg) ReplyErr(err error) *Msg       { return m.Reply(resData{Err: err}) }

type resData struct {
	Res interface{} `json:"res,omitempty"`
	Err error       `json:"err,omitempty"`
}

type TokMap struct {
	last int64
	m    map[int64]req
}

func (r *TokMap) Add(m *Msg) string {
	if r.m == nil {
		r.m = make(map[int64]req)
	}
	r.last++
	r.m[r.last] = req{m.From, m.Tok}
	return strconv.FormatInt(r.last, 16)
}

func (r *TokMap) Respond(m *Msg) error {
	if len(m.Tok) == 0 {
		return fmt.Errorf("empty response token %s", m.Subj)
	}
	id, err := strconv.ParseInt(m.Tok, 16, 64)
	if err != nil {
		return fmt.Errorf("invalid response token %s: %v", m.Tok, err)
	}
	req, ok := r.m[id]
	if !ok {
		return fmt.Errorf("no request with token %s", m.Tok)
	}
	n := *m
	n.Tok = req.tok
	req.Chan() <- &n
	delete(r.m, id)
	return nil
}

type req struct {
	Conn
	tok string
}
