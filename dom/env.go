package dom

import (
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

type Env struct {
	Par exp.Env
}

func NewEnv() *Env { return &Env{Par: extlib.Std} }

func FindEnv(env exp.Env) *Env {
	for ; env != nil; env = env.Parent() {
		if d, _ := env.(*Env); d != nil {
			return d
		}
	}
	return nil
}
func (e *Env) Parent() exp.Env { return e.Par }
func (e *Env) Lookup(s *exp.Sym, p cor.Path, eval bool) (lit.Val, error) {
	switch p.Plain() {
	case "project":
		return exp.NewSpecRef(projectSpec), nil
	case "schema":
		return exp.NewSpecRef(schemaSpec), nil
	case "model":
		return exp.NewSpecRef(modelSpec), nil
	case "elem":
		return exp.NewSpecRef(elemSpec), nil
	}
	return e.Par.Lookup(s, p, eval)
}
