// Package qry provides a way to describe and evaluate queries for local and external data.
package qry

import (
	"fmt"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Backend executes query jobs for the advertised dom schemas.
type Backend interface {
	Proj() *dom.Project
	Exec(*exp.Prog, *Job) (*exp.Lit, error)
}

// Doc is a query program environment that resolves query subjects and collects and tracks all jobs.
type Doc struct {
	Par exp.Env
	Backend
	Doms *dom.Schema

	All  []*Job
	Root []*Job
}

// New returns a new program environment to enable qry specs on the given backend.
func NewDoc(env exp.Env, bend Backend) *Doc {
	return &Doc{Par: env, Backend: bend, Doms: dom.Dom}
}

func FindDoc(env exp.Env) *Doc {
	for ; env != nil; env = env.Parent() {
		if d, _ := env.(*Doc); d != nil {
			return d
		}
	}
	return nil
}

// Add adds a job to the query document.
func (p *Doc) Add(j *Job) {
	p.All = append(p.All, j)
	if j.ParentJob() == nil {
		p.Root = append(p.Root, j)
	}
}

func (e *Doc) Parent() exp.Env { return e.Par }
func (e *Doc) Lookup(s *exp.Sym, p cor.Path, eval bool) (lit.Val, error) {
	if f := p.Fst(); f.Key != "" && strings.HasPrefix(s.Sym, f.Key) {
		switch c := s.Sym[0]; c {
		case '?', '*', '#':
			subj, err := e.Subject(s.Sym[1:])
			if err != nil {
				return nil, err
			}
			ps := []typ.Param{{Type: typ.Opt(typ.ElemTupl(typ.Exp))}, {}}
			sig := typ.Form(s.Sym, ps...)
			spec := &Spec{SpecBase: exp.SpecBase{Decl: sig}, Doc: e,
				Task: Task{Kind: Kind(c), Subj: subj}}
			return &exp.SpecRef{Spec: spec, Decl: sig}, nil
		}
	}
	return e.Par.Lookup(s, p, eval)
}

// Subj returns a subject from the first backend that provides ref or an error.
func (q *Doc) Subject(ref string) (*Subj, error) {
	switch ref[0] {
	case '.', '/', '$': // path subj
		return &Subj{Ref: ref, Bend: LitBackend{}}, nil
	}
	if q.Backend == nil {
		return nil, fmt.Errorf("no qry backend configured")
	}
	pr := q.Proj()
	switch ref {
	case "dom.model", "dom.schema", "dom.project":
		m := q.Doms.Model(ref[4:])
		return &Subj{Ref: ref, Bend: &DomBackend{pr}, Model: m}, nil
	}
	if m := pr.Model(ref); m != nil {
		s := &Subj{Ref: ref, Bend: q.Backend, Model: m}
		s.Type = m.Type()
		s.Fields = subjFields(s.Type)
		return s, nil
	}
	return nil, fmt.Errorf("no subj found for %q", ref)
}
