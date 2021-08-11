package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"xelf.org/xelf/bfr"
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
	// Raw is the message body as bytes.
	Raw []byte
	// Data is the typed body data and can be used to avoid serialization of internal messages.
	Data interface{}
}

// String returns the default string format of this message.
func (m *Msg) String() string {
	r := m.Raw
	if len(r) == 0 && m.Data != nil {
		r, _ = json.Marshal(m.Data)
	}
	return fmt.Sprintf("%s#%s\n%s", m.Subj, m.Tok, r)
}

// Parse parses str using the msg string format and returns a message or an error.
func Parse(str string) (*Msg, error) { return Read(strings.NewReader(str)) }

// Read parses reader r using the msg string format and returns a message or an error.
func Read(r io.Reader) (*Msg, error) {
	b := bfr.Get()
	defer bfr.Put(b)
	_, err := b.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	var subj, tok, raw []byte
	subj = b.Bytes()
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
