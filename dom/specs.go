package dom

import (
	"fmt"
	"strings"

	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

var domReg = &lit.Reg{}

type subSpecs = func(k string) exp.Spec

func domSpec(val interface{}, sig string, env bool, rs ext.Rules, sub subSpecs) *ext.NodeSpec {
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
	spec.Sub = sub
	return spec
}

var projectSpec = domSpec(&Project{}, "<form project name:sym tags:tupl?|exp @>", true, ext.Rules{
	Default: ext.Rule{
		Prepper: declsPrepper(schemaPrepper, ext.DynPrepper),
		Setter:  ext.ExtraSetter("extra"),
	},
}, func(k string) exp.Spec {
	if k == "load" {
		return load
	}
	return nil
})

var schemaSpec = domSpec(&Schema{}, "<form schema name:sym tags:tupl?|exp @>", true, ext.Rules{
	Default: ext.Rule{
		Prepper: declsPrepper(modelsPrepper, ext.DynPrepper),
		Setter:  ext.ExtraSetter("extra"),
	},
}, func(k string) exp.Spec {
	if k == ":" || k == "model" {
		return modelSpec
	}
	return nil
})

var modelSpec = domSpec(&Model{}, "<form model name:sym kind:typ tags:tupl?|exp @>", true, ext.Rules{
	Default: ext.Rule{
		Prepper: declsPrepper(elemsPrepper, ext.DynPrepper),
		Setter:  ext.ExtraSetter("extra"),
	},
}, func(k string) exp.Spec {
	if k == ":" || k == "elem" {
		return elemSpec
	}
	return nil
})

var bitRule = ext.Rule{Prepper: ext.BitsPrepper(bitConsts), Setter: ext.BitsSetter("bits")}
var elemSpec = domSpec(&Elem{}, "<form field name:sym type:typ tupl?|tag @>", false, ext.Rules{
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
}, nil)

func keySetter(key string) ext.KeySetter {
	return func(p *exp.Prog, n ext.Node, _ string, v lit.Val) error {
		return n.SetKey(key, v)
	}
}
func declsPrepper(decl, tag ext.KeyPrepper) ext.KeyPrepper {
	return func(p *exp.Prog, env exp.Env, n ext.Node, k string, arg exp.Exp) (lit.Val, error) {
		if k != "" && !cor.IsCased(k) && k[0] != '@' {
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
	mut, ok := aa.Value().(lit.Mut)
	if !ok || mut.Zero() {
		return nil, fmt.Errorf("not a project value %s", aa)
	}
	s := mut.Ptr().(*Schema)
	pro.Schemas = append(pro.Schemas, s)
	// here we can resolve type to previously defined schemas
	for _, m := range s.Models {
		err = reslDomRefs(m, s, pro)
		if err != nil {
			return nil, err
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
	mut, ok := aa.Value().(lit.Mut)
	if !ok || mut.Zero() {
		return nil, fmt.Errorf("not a model value %s", aa)
	}
	m := mut.Ptr().(*Model)
	m.Schema = s.Name
	s.Models = append(s.Models, m)
	// here we can resolve type references to the model itself and models in the same schema
	err = reslDomRefs(m, s, nil)
	if err != nil {
		return nil, err
	}
	p.Reg.SetRef(m.Qualified(), m.Type(), nil)
	return nil, nil
}
func elemsPrepper(p *exp.Prog, env exp.Env, n ext.Node, key string, arg exp.Exp) (_ lit.Val, err error) {
	m := n.Ptr().(*Model)
	el := &Elem{Name: key}
	k := m.Kind.Kind
	if k == 0 {
		k = knd.Obj
		m.Kind.Kind = k
	}
	switch k {
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
		if key != "" && key[0] == '@' {
			el, err = prepRefElem(key)
			if err != nil {
				return nil, err
			}
			if arg != nil {
				ta, err := p.Eval(env, arg)
				if err != nil {
					return nil, err
				}
				switch tv := ta.Val.(type) {
				case lit.Mut:
					n := tv.Ptr().(*Elem)
					n.Type = el.Type
					el = n
				}
			}
		} else if arg == nil {
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
				// special case to allow embedding consts
				if key == "" && k == knd.Obj && tv.Kind&(knd.Enum|knd.Bits) != 0 {
					name := typ.Name(tv)
					if idx := strings.LastIndexByte(name, '.'); idx >= 0 {
						name = name[idx+1:]
					}
					el.Name = cor.Cased(name)
				}
			}
		}
		if strings.HasSuffix(el.Name, "?") {
			el.Bits |= BitOpt
		}
	}
	m.Elems = append(m.Elems, el)
	return nil, nil
}
func reslDomRefs(m *Model, s *Schema, p *Project) (err error) {
	if m.Kind.Kind&knd.Obj == 0 {
		return nil
	}
	for _, el := range m.Elems {
		et := typ.ContEl(el.Type)
		if et.Kind&knd.Ref != 0 {
			err = reslDomRef(el, typ.Name(et), m, s, p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func reslDomRef(el *Elem, name string, m *Model, s *Schema, p *Project) (err error) {
	switch ps := strings.Split(name, "."); len(ps) {
	case 1:
		m = s.Model(cor.Keyed(name))
		return reslRefType(m, el)
	case 2:
		if ps[0] == "" { // .Field
			return reslRefField(m, ps[1], el)
		} // can be schema.Model or Model.Field
		if ps[0] == s.Name { // this.Model
			m = s.Model(cor.Keyed(ps[1]))
			return reslRefType(m, el)
		}
		m := s.Model(ps[0])
		if m != nil {
			return reslRefField(m, ps[1], el)
		}
		if p != nil {
			m = p.Model(cor.Keyed(name))
			return reslRefType(m, el)
		}
		return nil
	case 3:
		if ps[0] == "" { // ..Model
			if ps[1] == "" {
				m = s.Model(cor.Keyed(ps[2]))
				return reslRefType(m, el)
			}
		} else if ps[0] == s.Name || p != nil { // schema.Model.Field
			if ps[0] == s.Name {
				m = s.Model(cor.Keyed(ps[1]))
			} else {
				m = p.Model(cor.Keyed(name[:1+len(ps[0])+len(ps[1])]))
			}
			return reslRefField(m, ps[2], el)
		} else {
			return nil
		}
	case 4:
		if ps[0] == "" && ps[1] == "" { // ..Model.Field
			m = s.Model(cor.Keyed(ps[2]))
			return reslRefField(m, ps[3], el)
		}
	}
	return fmt.Errorf("unsupported dom reference %s", name)
}
func prepRefElem(key string) (*Elem, error) {
	ref := key[1:]
	var bits Bit
	if ref[len(ref)-1] == '?' {
		bits |= BitOpt
		ref = ref[:len(ref)-1]
	}
	return &Elem{Type: typ.Ref(ref), Bits: bits}, nil
}
func reslRefType(m *Model, el *Elem) error {
	if m == nil {
		return fmt.Errorf("model %s not found", el.Type)
	}
	// update type (usually container type)
	el.Type, _ = typ.Edit(el.Type, func(e *typ.Editor) (typ.Type, error) {
		if e.Kind&knd.Ref != 0 {
			return m.Type(), nil
		}
		return e.Type, nil
	})
	if el.Name == "" {
		el.Name = m.Name
	}
	return nil
}
func reslRefField(m *Model, key string, el *Elem) error {
	if m == nil {
		return fmt.Errorf("model %s not found", el.Type)
	}
	mt := m.Type()
	pb := mt.Body.(*typ.ParamBody)
	idx := pb.FindKeyIndex(cor.Keyed(key))
	if idx < 0 {
		return fmt.Errorf("key %s not found in %s", key, mt)
	}
	p := pb.Params[idx]
	if el.Name == "" {
		el.Name = m.Name
	}
	el.Ref = m.Qualified()
	// we assume id is usually the primary key and omit it from references
	if p.Key != "id" {
		el.Ref += "." + p.Key
	}
	if el.Name == "" {
		el.Name = m.Name
	}
	el.Type, _ = typ.Edit(el.Type, func(e *typ.Editor) (typ.Type, error) {
		if e.Kind&knd.Ref != 0 {
			return p.Type, nil
		}
		return e.Type, nil
	})
	return nil
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
