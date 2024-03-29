package dom

import (
	"fmt"

	"xelf.org/xelf/ast"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
	"xelf.org/xelf/typ"
)

// domSpec is a custom base for all dom specs. It reuses parts of the ext node spec.
type domSpec struct {
	exp.SpecBase
	ext.Rules
	nodeProv func(*exp.Prog) any
	declRule ext.KeyPrepper
	modHook  func(*exp.Prog, *mod.ModEnv, ext.Node)
	subSpec  exp.Spec
	dotHook  dotLookup
}

func (s *domSpec) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (_ exp.Exp, err error) {
	// get or create our custom node mod env
	var ne *NodeEnv
	if c.Env == nil {
		n, err := ext.NewNode(p.Reg, s.nodeProv(p))
		if err != nil {
			return nil, err
		}
		ne = &NodeEnv{Node: n, Sub: s.subSpec, dot: s.dotHook}
		if s.modHook != nil {
			ne.ModEnv = mod.NewModEnv(env, &p.File)
		} else {
			ne.ModEnv = &mod.ModEnv{Par: env}
		}
		c.Env = ne
	} else {
		ne = c.Env.(*NodeEnv)
	}
	// we already know the result type, check hint now
	if h != typ.Void {
		_, err := p.Sys.Unify(ne.Type(), h)
		if err != nil {
			return nil, err
		}
	}
	// evaluate all arguments
	for i, pa := range exp.SigArgs(c.Sig) {
		a := c.Args[i]
		if a == nil {
			if pa.Key == "kind" {
				a = exp.LitVal(typ.Type{Kind: knd.Obj})
			} else if pa.Key == "type" {
				a = exp.LitVal(typ.Type{Kind: knd.Void})
			} else {
				return nil, ast.ErrReslSpec(c.Src, fmt.Sprintf("nil arg %s %s", pa.Name, pa.Type), nil)
			}
		}
		if i == 0 { // name:sym in all dom specs
			a, err = p.Resl(c.Env, a, pa.Type)
			if err != nil {
				return nil, err
			}
			s, ok := a.(*exp.Lit)
			if !ok {
				return nil, fmt.Errorf("want a lit got %T", a)
			}
			ne.SetName(s.String())
			ne.SetKey(pa.Key, s.Val)
		} else if pa.Kind == knd.Typ { // model kind and elem type
			a, err = p.Resl(c.Env, a, pa.Type)
			if err != nil {
				return nil, err
			}
			rt := typ.Res(a.Type())
			if rt.Kind != knd.Typ {
				return nil, fmt.Errorf("expected resolved type got %#v %[1]T", a)
			}
			l := a.(*exp.Lit)
			t := l.Value().(typ.Type)
			t, err = p.Sys.Inst(exp.LookupType(c.Env), t)
			if err != nil {
				return nil, err
			}
			l.Val = t
			ne.SetKey(pa.Key, t)
			a = l
		} else if pa.Kind&knd.Tupl != 0 {
			tup := a.(*exp.Tupl)
			for j, d := range tup.Els {
				var key string
				if t, ok := d.(*exp.Tag); ok {
					key = t.Tag
					d = t.Exp
				}
				r := s.declRule
				istag := r == nil || key != "" && !cor.IsCased(key) && key[0] != '@'
				if istag {
					r = s.Rules.Eval
				}
				if d != nil {
					d, err = p.Resl(c.Env, d, typ.Void)
					if err != nil {
						return nil, err
					}
				}
				v, err := r(p, c.Env, ne.Node, key, d)
				if err != nil {
					return nil, err
				}
				if v != nil {
					tup.Els[j] = exp.LitVal(v)
				}
			}
		} else {
			return nil, fmt.Errorf("unexpected dom not param %s", pa.Name)
		}
		c.Args[i] = a
	}
	if s.modHook != nil {
		s.modHook(p, ne.ModEnv, ne.Node)
	}
	// keep the call for printing
	return c, ne.Publish()
}
func (s *domSpec) Eval(p *exp.Prog, c *exp.Call) (lit.Val, error) {
	return c.Env.(*NodeEnv).Node, nil
}

type NodeEnv struct {
	*mod.ModEnv
	ext.Node
	dot dotLookup
	Sub exp.Spec
}

func (e *NodeEnv) Lookup(s *exp.Sym, path cor.Path, eval bool) (lit.Val, error) {
	if path.Plain() == ":" && e.Sub != nil {
		return exp.NewSpecRef(e.Sub), nil
	}
	p := path
	if e.dot != nil {
		var ok bool
		if p, ok = exp.DotPath(p); ok {
			if v := e.dot(e, p); v != nil {
				s.Update(v.Type(), e, path)
				return v, nil
			}
			if s.Update(s.Res, e, path); !eval {
				return nil, nil
			}
			return nil, exp.ErrSymNotFound
		}
	}
	return e.ModEnv.Lookup(s, p, eval)
}

type any = interface{}
type dotLookup func(*NodeEnv, cor.Path) lit.Val

func prep(sig string, inst any, s *domSpec) *domSpec {
	s.SpecBase = exp.MustSpecBase(sig)
	p, err := lit.Proxy(domReg, inst)
	if err != nil {
		panic(err)
	}
	if s.nodeProv == nil {
		s.nodeProv = func(_ *exp.Prog) any {
			return p.New()
		}
	}
	rp := exp.SigRes(s.Decl)
	rp.Type = p.Type()
	return s
}
