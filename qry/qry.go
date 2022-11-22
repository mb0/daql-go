// Package qry provides a way to describe and evaluate queries for local and external data.
package qry

import (
	"context"
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Backend executes query jobs for the advertised dom schemas.
type Backend interface {
	Proj() *dom.Project
	Exec(*exp.Prog, *Job) (*exp.Lit, error)
}

// Qry is a context to execute queries on backends.
type Qry struct {
	Reg *lit.Reg
	Env exp.Env
	Backend
	doms *dom.Schema
}

// New returns a new query context with the program environment and backend.
func New(reg *lit.Reg, env exp.Env, bend Backend) *Qry {
	if reg == nil {
		reg = &lit.Reg{}
	}
	dom.SetupReg(reg)
	return &Qry{Backend: bend, Env: env, Reg: reg, doms: dom.Dom}
}

// Subj returns a subject from the first backend that provides ref or an error.
func (q *Qry) Subj(ref string) (*Subj, error) {
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
		m := q.doms.Model(ref[4:])
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

// Exec executes the given query str with arg and returns a value or an error.
func (q *Qry) Exec(ctx context.Context, str string, arg lit.Val) (*exp.Lit, error) {
	x, err := exp.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("parse qry %s error: %w", str, err)
	}
	return q.ExecExp(ctx, x, arg)
}

// ExecExp executes the given query expr with arg and returns a value or an error.
func (q *Qry) ExecExp(ctx context.Context, expr exp.Exp, arg lit.Val) (_ *exp.Lit, err error) {
	a, err := exp.NewProg(ctx, q.Reg, &Doc{Qry: q}).Run(expr, exp.LitVal(arg))
	if err != nil {
		return nil, fmt.Errorf("eval qry %s error: %w", expr, err)
	}
	return a, nil
}

// ExecAuto generates query from and saves the query result into a tagged go struct value pointer.
func (q *Qry) ExecAuto(ctx context.Context, pp interface{}, arg lit.Val) (lit.Mut, error) {
	x, err := ReflectQuery(q.Reg, pp)
	if err != nil {
		return nil, err
	}
	mut, err := q.Reg.Proxy(pp)
	if err != nil {
		return nil, err
	}
	el, err := q.ExecExp(ctx, x, arg)
	if err != nil {
		return nil, err
	}
	err = mut.Assign(el.Val)
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

func (e *Doc) Parent() exp.Env { return e.Env }
func (e *Doc) Lookup(s *exp.Sym, k string, eval bool) (exp.Exp, error) {
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
	return e.Env.Lookup(s, k, eval)
}
