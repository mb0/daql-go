package dom

import (
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

type Loader interface {
	Load(reg *lit.Reg, rel string) (exp.Exp, string, error)
}

type Env struct {
	Par exp.Env
	Loader
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
	}
	return e.Par.Lookup(s, k, eval)
}
