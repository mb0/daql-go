package qry

import (
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Kind indicates the query kind.
type Kind byte

const (
	KindOne   Kind = '?'
	KindMany  Kind = '*'
	KindCount Kind = '#'
)

// Ord holds key sort order information.
type Ord struct {
	Key  string
	Desc bool
	Subj bool
}

// Subj represents a query subject of a specific backend.
type Subj struct {
	Ref string
	Sel
	Model *dom.Model
	Bend  Backend
}

// Task describes a all the details for one query subj call.
type Task struct {
	Kind Kind
	*Subj
	Sel *Sel
	Res typ.Type
	Whr []exp.Exp
	Lim int64
	Off int64
	Ord []Ord
}

func (t *Task) Field(k string) (*Field, error) {
	f := t.Subj.Field(k)
	if f == nil && t.Sel != nil {
		f = t.Sel.Field(k)
	}
	if f == nil {
		return nil, fmt.Errorf("field %s not found in %s", k, t.Subj.Type)
	}
	return f, nil
}

// Job is the environment for a query spec and holds the task and its eventual result.
type Job struct {
	*Doc
	Env exp.Env
	*Task
	Val lit.Val
	Cur lit.Val
}

// FindJob returns a job environment that is env or one of its ancestors.
func FindJob(env exp.Env) *Job {
	for ; env != nil; env = env.Parent() {
		if par, ok := env.(*Job); ok {
			return par
		}
	}
	return nil
}

// ParentJob returns the parent job environment of this job or nil.
func (e *Job) ParentJob() *Job { return FindJob(e.Env) }
func (e *Job) Parent() exp.Env { return e.Env }
func (e *Job) Dyn() exp.Spec   { return e.Env.Dyn() }
func (e *Job) Resl(p *exp.Prog, s *exp.Sym, k string) (exp.Exp, error) {
	k, ok := exp.DotKey(k)
	if !ok {
		return e.Env.Resl(p, s, k)
	}
	f, err := e.Task.Field(k[1:])
	if err != nil {
		return nil, err
	}
	s.Type, s.Env, s.Rel = f.Type, e, k
	return s, nil
}
func (e *Job) Eval(p *exp.Prog, s *exp.Sym, k string) (*exp.Lit, error) {
	k, ok := exp.DotKey(k)
	if !ok {
		return e.Env.Eval(p, s, k)
	}
	_, err := e.Task.Field(k[1:])
	if err != nil {
		return nil, err
	}
	if e.Cur == nil && e.Val == nil {
		return nil, fmt.Errorf("job env unresolved %s in %s", s.Sym, e.Subj.Type)
	}
	v, err := lit.Select(e.Cur, k)
	if err != nil {
		return nil, err
	}
	return &exp.Lit{Res: typ.Lit, Val: v}, nil
}
