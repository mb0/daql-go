package dom

import (
	"fmt"
	"strings"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
	"xelf.org/xelf/typ"
)

var projectSpec = prep("<form@project name:sym tags:tupl?|exp @>", &Project{}, &domSpec{
	Rules:    ext.Rules{Default: ext.Rule{Setter: ext.ExtraSetter("extra")}},
	nodeProv: func(p *exp.Prog) any { return &Project{Extra: fileExtra(p.File.URL)} },
	declRule: schemaPrep,
	modHook:  func(me *mod.ModEnv, n ext.Node) { me.AddDecl("dom", n) },
})

func schemaPrep(p *exp.Prog, env exp.Env, n ext.Node, k string, e exp.Exp) (lit.Val, error) {
	a, err := p.Eval(env, e)
	if err != nil {
		return nil, err
	}
	s, ok := mutPtr(a).(*Schema)
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", a.Value())
	}
	pro := n.Ptr().(*Project)
	pro.Schemas = append(pro.Schemas, s)
	return a.Val, nil
}

var schemaSpec = prep("<form@schema name:sym tags:tupl?|exp @>", &Schema{}, &domSpec{
	Rules:    ext.Rules{Default: ext.Rule{Setter: ext.ExtraSetter("extra")}},
	nodeProv: func(p *exp.Prog) any { return &Schema{Extra: fileExtra(p.File.URL)} },
	declRule: modelPrep,
	modHook:  func(me *mod.ModEnv, n ext.Node) { me.AddDecl("dom", n) },
	subSpec:  modelSpec,
})

func modelPrep(p *exp.Prog, env exp.Env, n ext.Node, key string, e exp.Exp) (lit.Val, error) {
	a, err := p.Eval(env, e)
	if err != nil {
		return nil, err
	}
	m, ok := mutPtr(a).(*Model)
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", a.Value())
	}
	s := n.Ptr().(*Schema)
	qualifyModel(m, s.Name)
	s.Models = append(s.Models, m)
	ne := env.(*NodeEnv)
	ne.AddDecl(m.Name, m.Type())
	return a.Val, nil
}

var modelSpec = prep("<form@model name:sym kind:typ tags:tupl?|exp @dom.Model>", &Model{}, &domSpec{
	Rules: ext.Rules{
		Key: map[string]ext.Rule{
			"idx":  idxRule,
			"uniq": idxRule,
		},
		Default: ext.Rule{Setter: ext.ExtraSetter("extra")},
	},
	declRule: elemsPrepper,
	subSpec:  elemSpec,
	dotHook: func(ne *NodeEnv, k string) lit.Val {
		m := ne.Node.Ptr().(*Model)
		for _, el := range m.Elems {
			if el.Name == k || el.Key() == k {
				t := el.Type
				t.Ref = m.Name + "." + el.Name
				return t
			}
		}
		return nil
	},
})

var elemSpec = prep("<form@elem name:sym type:typ tupl?|tag @>", &Elem{}, &domSpec{
	Rules: ext.Rules{
		Key: map[string]ext.Rule{
			"opt":  bitRule,
			"pk":   bitRule,
			"idx":  bitRule,
			"uniq": bitRule,
			"asc":  bitRule,
			"desc": bitRule,
			"auto": bitRule,
			"ro":   bitRule,
		},
		Default: ext.Rule{Setter: ext.ExtraSetter("extra")},
	},
})

func fileExtra(url string) *lit.Dict {
	if url == "" {
		return nil
	}
	x := new(lit.Dict)
	x.SetKey("file", lit.Str(url))
	return x
}

func idxAppender(p *exp.Prog, env exp.Env, n ext.Node, s string, arg exp.Exp) (_ lit.Val, err error) {
	m := n.Ptr().(*Model)
	switch a := arg.(type) {
	case nil:
		return nil, fmt.Errorf("index %s with nil arg", s)
	case *exp.Lit:
		switch av := a.Val.(type) {
		case *lit.Vals:
			if m.Object == nil {
				m.Object = &Object{}
			}
			idx := &Index{Unique: s == "uniq"}
			for _, v := range *av {
				s, err := lit.ToStr(v)
				if err != nil {
					return nil, err
				}
				idx.Keys = append(idx.Keys, string(s))
			}
			m.Object.Indices = append(m.Object.Indices, idx)
		case lit.Char:
			if m.Object == nil {
				m.Object = &Object{}
			}
			idx := &Index{Unique: s == "uniq", Keys: []string{av.String()}}
			m.Object.Indices = append(m.Object.Indices, idx)
		default:
			return nil, fmt.Errorf("index %s unexpected value %T", s, av)
		}
		return nil, nil
	}
	return nil, fmt.Errorf("index %s unexpected arg %T", s, arg)
}

func noopSetter(p *exp.Prog, n ext.Node, key string, v lit.Val) error { return nil }

var idxRule = ext.Rule{Prepper: idxAppender, Setter: noopSetter}

var bitRule = ext.Rule{Prepper: ext.BitsPrepper(bitConsts), Setter: ext.BitsSetter("bits")}

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
		if arg == nil {
			if key == "" || key[0] != '@' {
				return nil, fmt.Errorf("invalid element")
			}
			ref := key[1:]
			fst, _, _ := strings.Cut(ref, ".")
			if fst == "" {
				fst = m.Name
			}
			el.Name = fst
			arg = exp.LitVal(typ.Ref(ref))
		}
		ta, err := p.Eval(env, arg)
		if err != nil {
			return nil, err
		}
		switch tv := ta.Val.(type) {
		case lit.Mut:
			n, ok := tv.Ptr().(*Elem)
			if !ok {
				return nil, fmt.Errorf("expected *Elem got %s", tv.Value())
			}
			if n.Name == "" {
				n.Name = el.Name
			}
			el = n
		case typ.Type:
			el.Type = tv
			if key == "" && k != knd.Func {
				if tv.Ref == "" {
					return nil, fmt.Errorf("must be named type got %s", tv)

				}
				if tv.Kind&knd.Data != knd.Obj {
					el.Name = refElemName(tv.Ref)
				}
			}
		}
		if k == knd.Obj && el.Name == "ID" && len(m.Elems) == 0 {
			el.Bits |= BitPK
			el.Type.Ref = m.Name + ".ID"
		}
		if strings.HasSuffix(el.Name, "?") {
			el.Bits |= BitOpt
		}
	}
	m.Elems = append(m.Elems, el)
	return nil, nil
}
func qualifyModel(m *Model, sch string) error {
	if m.Schema != "" {
		return fmt.Errorf("model %s already part of schema %s", m.Name, m.Schema)
	}
	m.Schema = sch
	if m.Kind.Kind&(knd.Obj|knd.Func) != 0 {
		pref := m.Name + "."
		prep := sch + "."
		for _, el := range m.Elems {
			el.Type, _ = typ.Edit(el.Type, func(e *typ.Editor) (typ.Type, error) {
				if strings.HasPrefix(e.Ref, pref) {
					e.Ref = prep + e.Ref
				}
				return e.Type, nil
			})
		}
	}
	return nil
}
func refElemName(ref string) string {
	idx := strings.IndexByte(ref, '.')
	if idx < 0 {
		return ref
	}
	snd := ref[idx+1:]
	idx = strings.IndexByte(snd, '.')
	if idx < 0 {
		return snd
	}
	return snd[:idx]
}

func mutPtr(l *exp.Lit) interface{} {
	mut, ok := l.Value().(lit.Mut)
	if !ok || mut.Zero() {
		return nil
	}
	return mut.Ptr()
}
