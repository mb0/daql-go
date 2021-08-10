// Package domtest has default schemas and helpers for testing.
package domtest

import (
	"fmt"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/xelf/lit"
)

type Fixture struct {
	dom.Project
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
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
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

func (f *Fixture) Keys() []string { return f.Fix.Keys() }
func (f *Fixture) Close() error   { return nil }
func (f *Fixture) Iter(key string) (*idxrIter, error) {
	l, _ := f.Fix.Key(key)
	idxr, ok := l.(lit.Idxr)
	if !ok {
		return nil, fmt.Errorf("want idxr got %T", l)
	}
	return &idxrIter{idxr, 0}, nil
}

type idxrIter struct {
	lit.Idxr
	idx int
}

func (it *idxrIter) Close() error { return nil }

func (it *idxrIter) Scan() (lit.Val, error) {
	v, err := it.Idx(it.idx)
	it.idx++
	return v, err
}
