package qry

import (
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/cor"
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
	Val *exp.Lit
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
func (e *Job) Lookup(s *exp.Sym, p cor.Path, eval bool) (lit.Val, error) {
	p, ok := exp.DotPath(p)
	if !ok {
		return e.Env.Lookup(s, p, eval)
	}
	f, err := e.Task.Field(p.Fst().Key) // TODO
	if err != nil {
		return nil, err
	}
	if s.Update(f.Type, e, p); !eval {
		return nil, nil
	}
	if e.Cur == nil && e.Val == nil {
		return nil, fmt.Errorf("job env unresolved %s in %s", s.Sym, e.Subj.Type)
	}
	return lit.SelectPath(e.Cur, p)
}
