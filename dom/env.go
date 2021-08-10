package dom

import (
	"fmt"
	"io"
	"os"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/lib"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

func OpenSchema(name string, pro *Project) (s *Schema, _ error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSchema(f, name, pro)
}

func ReadSchema(r io.Reader, name string, pro *Project) (s *Schema, _ error) {
	reg := &lit.Reg{}
	reg.AddFrom(domReg)
	if pro == nil {
		pro = &Project{}
	}
	n, err := ext.NewNode(reg, pro)
	if err != nil {
		return nil, err
	}
	x, err := exp.Read(reg, r, name)
	if err != nil {
		return nil, err
	}
	env := &ext.NodeEnv{Par: NewEnv(nil), Node: n}
	l, err := exp.EvalExp(reg, env, x)
	if err != nil {
		return nil, err
	}
	mut, ok := l.Value().(lit.Mut)
	if ok {
		s, ok = mut.Ptr().(*Schema)
	}
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", l.Value())
	}
	return s, nil
}

type Loader interface {
	Load(rel string) (exp.Exp, string, error)
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
