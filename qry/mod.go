package qry

import (
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/ast"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
	"xelf.org/xelf/typ"
)

// Mod is the xelf module source for this package that encapsulates the qry setup.
var Mod *mod.Src

func init() {
	Mod = mod.Registry.Register(&mod.Src{
		Rel:   "daql/qry",
		Loc:   mod.Loc{URL: "xelf:daql/qry"},
		Setup: modSetup,
	})
}

func modSetup(prog *exp.Prog, s *mod.Src) (f *mod.File, err error) {
	// ensure the dom module is loaded
	if f := prog.Files[dom.Mod.URL]; f == nil {
		le := mod.FindLoaderEnv(prog.Root)
		if le == nil {
			return nil, fmt.Errorf("no loader env found")
		}
		f, err = le.LoadFile(prog, &dom.Mod.Loc)
		if err != nil {
			return nil, err
		}
		prog.File.Refs = append(prog.File.Refs, f.Refs...)
	}
	// ensure a qry doc
	doc := FindDoc(prog.Root)
	if doc == nil {
		// leave backend empty here
		doc = &Doc{Qry: &Qry{Env: prog.Root, Reg: prog.Reg, doms: dom.Dom}}
		prog.Root = doc
	}
	f = &exp.File{URL: s.URL}
	me := mod.NewModEnv(prog, f, ast.Src{})
	me.SetName("qry")
	me.AddDecl("bend", bend)
	return f, nil
}

var bend = &bendSpec{exp.MustSpecBase("<form@qry.bend uri:str pro?:@dom.Project none>")}

type bendSpec struct {
	exp.SpecBase
}

func (s *bendSpec) Value() lit.Val { return s }

func (s *bendSpec) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.Exp, error) {
	if c.Env != nil {
		return c, nil
	}
	_, err := s.SpecBase.Resl(p, env, c, h)
	if err != nil {
		return nil, err
	}
	a, err := p.Eval(c.Env, c.Args[0])
	if err != nil {
		return nil, err
	}
	uri := a.Val.String()
	var pro *dom.Project
	if len(c.Args) > 1 && c.Args[1] != nil {
		a, err = p.Eval(c.Env, c.Args[1])
		if err != nil {
			return nil, err
		}
		pro = a.Value().(lit.Mut).Ptr().(*dom.Project)
	} else {
		if f := p.Files[dom.Mod.URL]; f != nil {
			m := f.Refs.Find("dom")
			if m == nil {
				return nil, fmt.Errorf("dom module not found")
			}
			a, err := lit.SelectKey(m.Decl.Val, "projects")
			if err == nil {
				l := a.(*lit.List)
				if n := len(l.Vals); n > 0 {
					pro = l.Vals[n-1].(lit.Mut).Ptr().(*dom.Project)
				}
			}
		}
	}
	if pro == nil {
		return nil, fmt.Errorf("no project found for %s", uri)
	}
	doc := FindDoc(p.Root)
	if doc == nil {
		return nil, fmt.Errorf("no doc env found")
	}
	if doc.Backend != nil {
		return nil, fmt.Errorf("backend already set")
	}
	loc := mod.ParseLoc(string(uri))
	prov := LoadProvider(loc.Proto())
	if prov == nil {
		return nil, fmt.Errorf("no backend found for %s", uri)
	}
	bend, err := prov.Provide(string(uri), pro)
	if err != nil {
		return nil, err
	}
	doc.Qry.Backend = bend
	return c, nil
}

func (s *bendSpec) Eval(p *exp.Prog, c *exp.Call) (*exp.Lit, error) {
	return exp.LitVal(lit.Null{}), nil
}
