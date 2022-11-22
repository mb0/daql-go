package dom

import (
	"fmt"
	"strings"

	"xelf.org/xelf/ast"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
)

var domReg = &lit.Reg{}

// Mod is the xelf module source for this package that encapsulates the setup required to work
// with dom specs and gives access to schemas and model beyond their type.
var Mod *mod.Src

var Dom *Schema

func init() {
	var err error
	Dom, err = ReadSchema(domReg, strings.NewReader(RawSchema()), "go:xelf.org/daql/dom/dom.xelf")
	if err != nil {
		panic(fmt.Errorf("could not read dom schema: %w", err))
	}
	Mod = mod.Registry.Register(&mod.Src{
		Rel:   "daql/dom",
		Loc:   mod.Loc{URL: "xelf:daql/dom"},
		Setup: modSetup,
	})
}

func SetupReg(reg *lit.Reg) {
	reg.AddFrom(domReg)
}

func modSetup(prog *exp.Prog, s *mod.Src) (*mod.File, error) {
	// register dom proxies
	SetupReg(prog.Reg)
	// ensure a dom env
	if de := FindEnv(prog.Root); de == nil {
		prog.Root = &Env{Par: prog.Root}
	}
	f := &exp.File{URL: s.URL}
	me := mod.NewModEnv(prog, f, ast.Src{})
	me.SetName("dom")
	me.AddDecl("dom", lit.MustProxy(prog.Reg, Dom))
	for _, m := range Dom.Models {
		me.AddDecl(m.Name, m.Type())
	}
	me.AddDecl("projects", &lit.List{El: Dom.Model("project").Type()})
	return f, nil
}
