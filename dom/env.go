package dom

import (
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
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
func (e *Env) Lookup(s *exp.Sym, k string, eval bool) (exp.Exp, error) {
	switch s.Sym {
	case "project":
		return exp.LitVal(projectSpec), nil
	case "schema":
		return exp.LitVal(schemaSpec), nil
	case "model":
		return exp.LitVal(modelSpec), nil
	case "elem":
		return exp.LitVal(elemSpec), nil
	}
	return e.Par.Lookup(s, k, eval)
}
