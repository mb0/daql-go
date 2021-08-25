// Package ses provides an abstraction over varying, http session authentication systems.
// The basic idea is that we receive some kind of token in an http request either way.
// The token contains at least a session identifier that we can use to lookup session data.
package ses

import (
	"fmt"
	"net/http"
)

// Session decorates the user session data with additional fields.
type Session struct {
	Data
	IsNew    bool
	IsCookie bool
}

// Data is the user defined session data that is often coupled to a specific application and store.
type Data interface {

	// ID returns the session id.
	ID() string

	// Tok returns the data to be encoded in the token.
	// This is usually just the session id but can be any string encoded value.
	Tok() string

	// User returns the user or account id if authenticated.
	User() string
}

// Store provides access to often persisted session data.
type Store interface {

	// New creates and returns session data that must at least have a unique id or an error.
	New() (Data, error)

	// Get returns session data for the given token data or an error.
	Get(td string) (Data, error)

	// Save persist the given session data or returns an error.
	Save(d Data, isnew bool) error

	// Delete deletes the session data for the given token data or returns an error.
	Delete(td string) error
}

var errNoToken = fmt.Errorf("no session token")

// TokenReader can read a token from a http request.
type TokenReader interface {
	ReadToken(*http.Request) string
}

// TokenWriter can write a token to a http response, usually cookies.
// The token should be cleared when called with an empty token string.
type TokenWriter interface {
	WriteToken(http.ResponseWriter, string)
}

// TokenCodec and decode and encode a token to the underlying token data.
// Data is used as string in this context, but anything can be encode as string.
// You can wrap github.com/gorilla/securecookie as a simple an proven token codec.
// Or you can check your claims from any kind of bearer token you want to use.
type TokenCodec interface {
	DecodeToken(tok string) (td string, err error)
	EncodeToken(td string) (tok string, err error)
}

// Config represent a session token configuration with multiple codecs and helpers.
type Config struct {
	// TokenReader reads tokens from requests. Some readers do also implement token write.
	TokenReader
	// Codec is codec list that is tried from first to last both for decoding and encoding.
	// This allows seamless key rotations for session encodings. We try older codecs to fail
	// gracefully if a newly introduced codec cannot encode the token data.
	Codec []TokenCodec
}

// HeaderReader is a token reader that reads a named http request header
type HeaderReader string

func (tr HeaderReader) ReadToken(r *http.Request) string { return r.Header.Get(string(tr)) }
