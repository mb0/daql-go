// Package domtest has default schemas and helpers for testing.
package domtest

import (
	"fmt"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/daql/mig"
	"xelf.org/xelf/lit"
)

type Fixture struct {
	dom.Project
	*mig.Version
	Fix *lit.Dict
	reg lit.Reg
}

func New(raw, fix string) (*Fixture, error) {
	res := &Fixture{}
	s, err := dom.ReadSchema(strings.NewReader(raw), "", nil)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	res.Project.Schemas = append(res.Project.Schemas, s)
	mani, err := mig.Manifest{}.Update(&res.Project)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	res.Version = mani.First()
	val, err := lit.Read(&lit.Reg{}, strings.NewReader(fix), "")
	if err != nil {
		return nil, fmt.Errorf("fixture: %w", err)
	}
	res.Fix = val.(*lit.Dict)
	return res, nil
}

func Must(pro *Fixture, err error) *Fixture {
	if err != nil {
		panic(err)
	}
	return pro
}

func (f *Fixture) Vers() *mig.Version { return f.Version }
func (f *Fixture) Keys() []string     { return f.Fix.Keys() }
func (f *Fixture) Close() error       { return nil }
func (f *Fixture) Stream(key string) (mig.Stream, error) {
	l, _ := f.Fix.Key(key)
	idxr, ok := l.(lit.Idxr)
	if !ok {
		return nil, fmt.Errorf("want idxr got %T", l)
	}
	return mig.NewLitStream(idxr), nil
}

var _ mig.Dataset = (*Fixture)(nil)
