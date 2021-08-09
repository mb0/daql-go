package dom

import (
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

var domReg = &lit.Reg{}

func domSpec(val interface{}, sig string, env bool, rs ext.Rules) *ext.NodeSpec {
	n, err := ext.NewNode(domReg, val)
	if err != nil {
		panic(err)
	}
	s, err := typ.Parse(sig)
	if err != nil {
		panic(err)
	}
	exp.SigRes(s).Type = n.Type()
	spec := ext.NewNodeSpec(s, n, rs)
	spec.Env = env
	return spec
}

var projectSpec = func() *ext.NodeSpec {
	return domSpec(&Project{}, "<form project name:sym tags:tupl?|exp @>", true, ext.Rules{
		Default: ext.Rule{
			Prepper: declsPrepper(schemaPrepper, ext.DynPrepper),
			Setter:  ext.ExtraSetter("extra"),
		},
	})
}()

var schemaSpec = func() *ext.NodeSpec {
	spec := domSpec(&Schema{}, "<form schema name:sym tags:tupl?|exp @>", true, ext.Rules{
		Default: ext.Rule{
			Prepper: declsPrepper(modelsPrepper, ext.DynPrepper),
			Setter:  ext.ExtraSetter("extra"),
		},
	})
	spec.Sub = func(k string) exp.Spec {
		if k == ":" {
			return modelSpec
		}
		return nil
	}
	return spec
}()

var modelSpec = func() *ext.NodeSpec {
	spec := domSpec(&Model{}, "<form model name:sym kind:typ tags:tupl?|exp @>", true, ext.Rules{
		Default: ext.Rule{
			Prepper: declsPrepper(elemsPrepper, ext.DynPrepper),
			Setter:  ext.ExtraSetter("extra"),
		},
	})
	spec.Sub = func(k string) exp.Spec {
		if k == ":" {
			return elemSpec
		}
		return nil
	}
	return spec
}()

var elemSpec = func() *ext.NodeSpec {
	bitRule := ext.Rule{Prepper: ext.BitsPrepper(bitConsts), Setter: ext.BitsSetter("bits")}
	return domSpec(&Elem{}, "<form field name:sym type:typ tupl?|tag @>", false, ext.Rules{
		Key: map[string]ext.Rule{
			"opt":  bitRule,
			"pk":   bitRule,
			"idx":  bitRule,
			"uniq": bitRule,
			"ordr": bitRule,
			"auto": bitRule,
			"ro":   bitRule,
		},
		Default: ext.Rule{Setter: ext.ExtraSetter("extra")},
	})
}()

func keySetter(key string) ext.KeySetter {
	return func(p *exp.Prog, n ext.Node, _ string, v lit.Val) error {
		return n.SetKey(key, v)
	}
}
func declsPrepper(decl, tag ext.KeyPrepper) ext.KeyPrepper {
	return func(p *exp.Prog, env exp.Env, n ext.Node, k string, arg exp.Exp) (lit.Val, error) {
		if k != "" && !cor.IsCased(k) {
			return tag(p, env, n, k, arg)
		}
		return decl(p, env, n, k, arg)
	}
}
func schemaPrepper(p *exp.Prog, env exp.Env, n ext.Node, _ string, arg exp.Exp) (lit.Val, error) {
	pro := n.Ptr().(*Project)
	aa, err := p.Eval(env, arg)
	if err != nil {
		return nil, err
	}
	if v := aa.Value(); !v.Zero() {
		if mut, ok := v.(lit.Mut); ok {
			s := mut.Ptr().(*Schema)
			pro.Schemas = append(pro.Schemas, s)
		}
	}
	return nil, nil
}
func modelsPrepper(p *exp.Prog, env exp.Env, n ext.Node, _ string, arg exp.Exp) (lit.Val, error) {
	s := n.Ptr().(*Schema)
	aa, err := p.Eval(env, arg)
	if err != nil {
		return nil, err
	}
	if v := aa.Value(); !v.Zero() {
		if mut, ok := v.(lit.Mut); ok {
			m := mut.Ptr().(*Model)
			m.Schema = s.Name
			p.Reg.SetRef(m.Qualified(), m.Type(), mut)
			s.Models = append(s.Models, m)
		}
	}
	return nil, nil
}
func elemsPrepper(p *exp.Prog, env exp.Env, n ext.Node, key string, arg exp.Exp) (lit.Val, error) {
	m := n.Ptr().(*Model)
	el := &Elem{Name: key}
	switch k := m.Kind.Kind; k {
	case knd.Bits, knd.Enum:
		if arg == nil {
			if k == knd.Bits {
				el.Val = 1 << len(m.Elems)
			} else {
				el.Val = int64(len(m.Elems) + 1)
			}
		} else {
			ta, err := p.Eval(env, arg)
			if err != nil {
				return nil, err
			}
			switch tv := ta.Val.(type) {
			case lit.Mut:
				n := tv.Ptr().(*Elem)
				if n.Name == "" {
					n.Name = el.Name
				}
				el = n
			case lit.Val:
				val, err := lit.ToInt(ta.Val)
				if err != nil {
					return nil, err
				}
				el.Val = int64(val)
			}
		}
	case knd.Obj, knd.Func:
		if arg == nil {
			el.Type = typ.Any
		} else {
			ta, err := p.Eval(env, arg)
			if err != nil {
				return nil, err
			}
			switch tv := ta.Val.(type) {
			case lit.Mut:
				n := tv.Ptr().(*Elem)
				if n.Name == "" {
					n.Name = el.Name
				}
				el = n
			case typ.Type:
				el.Type = tv
			}
		}
	}
	m.Elems = append(m.Elems, el)
	return nil, nil
}
func forEach(arg exp.Exp, f func(exp.Exp) error) error {
	if tup, ok := arg.(*exp.Tupl); ok {
		for _, el := range tup.Els {
			if err := f(el); err != nil {
				return err
			}
		}
		return nil
	}
	return f(arg)
}
