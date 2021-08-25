// Package qry provides a way to describe and evaluate queries for local and external data.
package qry

import (
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Backend executes query jobs for the advertised dom schemas.
type Backend interface {
	Proj() *dom.Project
	Exec(*exp.Prog, *Job) (lit.Val, error)
}

// Backends is a list of backends that falls back on a lit backend for path subjects.
type Backends []Backend

// Subj returns a subject from the first backend that provides ref or an error.
func (bs Backends) Subj(ref string) (*Subj, error) {
	switch ref[0] {
	case '.', '/', '$': // path subj
		s := Subj{Ref: ref, Bend: &LitBackend{}}
		return &s, nil
	}
	for _, b := range bs {
		if m := b.Proj().Model(ref); m != nil {
			s := &Subj{Ref: ref, Bend: b, Model: m}
			s.Type = m.Type()
			s.Fields = subjFields(s.Type)
			return s, nil
		}
	}
	return nil, fmt.Errorf("no subj found for %q", ref)
}

// Qry is a context to execute queries on backends.
type Qry struct {
	Reg *lit.Reg
	Env exp.Env
	Backends
}

// New returns a new query context with the program environment and backends.
func New(reg *lit.Reg, env exp.Env, bends ...Backend) *Qry {
	if reg == nil {
		reg = &lit.Reg{}
	}
	return &Qry{Backends: Backends(bends), Env: env, Reg: reg}
}

// Exec executes the given query str with arg and returns a value or an error.
func (q *Qry) Exec(str string, arg lit.Val) (lit.Val, error) {
	x, err := exp.Parse(q.Reg, str)
	if err != nil {
		return nil, fmt.Errorf("parse qry %s error: %w", str, err)
	}
	return q.ExecExp(x, arg)
}

// ExecExp executes the given query expr with arg and returns a value or an error.
func (q *Qry) ExecExp(expr exp.Exp, arg lit.Val) (_ lit.Val, err error) {
	var env exp.Env = &Doc{Qry: q}
	if arg != nil {
		env = &exp.ArgEnv{Par: env, Typ: arg.Type(), Val: arg}
	}
	a, err := exp.EvalExp(q.Reg, env, expr)
	if err != nil {
		return nil, fmt.Errorf("eval qry %s error: %w", expr, err)
	}
	return a.Val, nil
}

// ExecAuto generates query from and saves the query result into a tagged go struct value pointer.
func (q *Qry) ExecAuto(pp interface{}, arg lit.Val) (lit.Mut, error) {
	x, err := ReflectQuery(q.Reg, pp)
	if err != nil {
		return nil, err
	}
	mut, err := q.Reg.Proxy(pp)
	if err != nil {
		return nil, err
	}
	el, err := q.ExecExp(x, arg)
	if err != nil {
		return nil, err
	}
	err = mut.Assign(el)
	if err != nil {
		return nil, err
	}
	return mut, nil
}

// Doc is an query program environment that collects and tracks all jobs.
type Doc struct {
	*Qry
	All  []*Job
	Root []*Job
}

// Add adds a job to the query document.
func (p *Doc) Add(j *Job) {
	p.All = append(p.All, j)
	if j.ParentJob() == nil {
		p.Root = append(p.Root, j)
	}
}

func (e *Doc) Parent() exp.Env { return e.Env }
func (e *Doc) Resl(p *exp.Prog, s *exp.Sym, k string) (exp.Exp, error) {
	switch c := s.Sym[0]; c {
	case '?', '*', '#':
		subj, err := e.Subj(s.Sym[1:])
		if err != nil {
			return s, err
		}
		sig := typ.Form(s.Sym, typ.P("", typ.Opt(typ.ElemTupl(typ.Exp))), typ.Param{})
		spec := &Spec{SpecBase: exp.SpecBase{Decl: sig}, Doc: e, Task: Task{Kind: Kind(c), Subj: subj}}
		return &exp.Lit{Res: sig.Type(), Val: spec, Src: s.Src}, nil
	}
	return e.Env.Resl(p, s, k)
}
func (e *Doc) Eval(p *exp.Prog, s *exp.Sym, k string) (*exp.Lit, error) {
	return e.Env.Eval(p, s, k)
}
