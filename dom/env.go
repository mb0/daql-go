package dom

import (
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib"
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

func NewEnv(par exp.Env) *Env {
	if par == nil {
		par = exp.Builtins(make(lib.Specs).AddMap(extlib.Std))
	}
	return &Env{Par: par}
}

func FindEnv(env exp.Env) *Env {
	for env != nil {
		if d, ok := env.(*Env); ok {
			return d
		}
		env = env.Parent()
	}
	return nil
}
func (e *Env) Parent() exp.Env { return e.Par }
func (e *Env) Dyn() exp.Spec   { return e.Par.Dyn() }
func (e *Env) Resl(p *exp.Prog, s *exp.Sym, k string) (exp.Exp, error) {
	var def exp.Spec
	switch s.Sym {
	case "project":
		def = projectSpec
	case "schema":
		def = schemaSpec
	}
	if def != nil {
		return &exp.Lit{Res: def.Type(), Val: def}, nil
	}
	return e.Par.Resl(p, s, k)
}
func (e *Env) Eval(p *exp.Prog, s *exp.Sym, k string) (*exp.Lit, error) {
	return e.Par.Eval(p, s, k)
}
