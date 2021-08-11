package mig

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"xelf.org/xelf/ast"
	"xelf.org/xelf/lit"
)

// Stream is an iterator for a possibly large sequence of object literal data.
//
// This abstraction allows us to choose an appropriate implementation for any situation, without
// being forced to load all the data into memory all at once.
type Stream interface {
	Scan() (lit.Val, error)
	Close() error
}

// OpenFileStream opens a file and returns a new stream or an error.
func OpenFileStream(path string) (Stream, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewFileStream(f, path, gzipped(path))
}

// NewFileStream creates and returns a new stream for the named reader f or an error.
func NewFileStream(f io.ReadCloser, name string, gzipped bool) (Stream, error) {
	if !gzipped {
		return &fileStream{f: f, lex: ast.NewLexer(f, name)}, nil
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &fileStream{f: f, gz: gz, lex: ast.NewLexer(gz, name)}, nil
}

// WriteStream writes stream to writer w or returns an error.
func WriteStream(it Stream, w io.Writer) error {
	enc := json.NewEncoder(w)
	for {
		l, err := it.Scan()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		err = enc.Encode(l)
		if err != nil {
			return err
		}
	}
	return nil
}

type fileStream struct {
	f   io.ReadCloser
	gz  *gzip.Reader
	lex *ast.Lexer
}

func (it *fileStream) Close() error {
	if it.gz != nil {
		it.gz.Close()
	}
	return it.f.Close()
}

func (it *fileStream) Scan() (lit.Val, error) {
	tr, err := ast.Scan(it.lex)
	if err != nil {
		return nil, err
	}
	reg := &lit.Reg{}
	v, err := reg.ParseVal(tr)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func gzipped(path string) bool { return strings.HasSuffix(path, ".gz") }
