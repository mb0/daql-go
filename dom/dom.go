package dom

import (
	"fmt"
	"strings"

	"xelf.org/xelf/knd"
	"xelf.org/xelf/typ"
)

func (s *Schema) Key() string { return strings.ToLower(s.Name) }
func (m *Model) Key() string  { return strings.ToLower(m.Name) }
func (e *Elem) Key() string   { return strings.ToLower(e.Name) }
func (m *Model) Params() []typ.Param {
	res := make([]typ.Param, 0, len(m.Elems))
	for _, el := range m.Elems {
		res = append(res, typ.P(el.Name, el.Type))
	}
	return res
}
func (m *Model) Consts() []typ.Const {
	res := make([]typ.Const, 0, len(m.Elems))
	for _, el := range m.Elems {
		res = append(res, typ.C(el.Name, el.Val))
	}
	return res
}
func (m *Model) Type() typ.Type {
	t := m.Kind
	switch t.Kind {
	case knd.Bits, knd.Enum:
		t.Body = &typ.ConstBody{Name: m.Qualified(), Consts: m.Consts()}
	case knd.Func:
		t.Body = &typ.ParamBody{Name: m.Qualified(), Params: m.Params()}
	case knd.Obj:
		res := make([]typ.Param, 0, len(m.Elems)+8)
		for _, el := range m.Elems {
			if el.Name == "" && el.Type.Kind&knd.Strc != 0 {
				res = flatParams(el.Type, res)
			} else {
				res = append(res, typ.P(el.Name, el.Type))
			}
		}
		t.Body = &typ.ParamBody{Name: m.Qualified(), Params: res}
	}
	return t
}

func flatParams(strc typ.Type, ps []typ.Param) []typ.Param {
	b := strc.Body.(*typ.ParamBody)
	for _, p := range b.Params {
		if p.Key != "" {
			ps = append(ps, p)
		} else {
			ps = flatParams(p.Type, ps)
		}
	}
	return ps
}

type Node interface {
	Qualified() string
}

func (m *Model) Qual() string      { return m.Schema }
func (m *Model) Qualified() string { return fmt.Sprintf("%s.%s", m.Schema, m.Name) }

func (s *Schema) Qualified() string { return s.Key() }

func (p *Project) Qualified() string { return fmt.Sprintf("_%s", p.Name) }

// Schema returns a schema for key or nil.
func (p *Project) Schema(key string) *Schema {
	if p != nil {
		for _, s := range p.Schemas {
			if s.Name == key {
				return s
			}
		}
	}
	return nil
}

// Model returns a model for the qualified key or nil.
func (p *Project) Model(key string) *Model {
	if p != nil {
		split := strings.SplitN(key, ".", 2)
		if len(split) == 2 {
			return p.Schema(split[0]).Model(split[1])
		}
		for _, s := range p.Schemas {
			if m := s.Model(key); m != nil {
				return m
			}
		}
	}
	return nil
}

// Model returns a model for key or nil.
func (s *Schema) Model(key string) *Model {
	if s != nil {
		for _, m := range s.Models {
			if m.Key() == key {
				return m
			}
		}
	}
	return nil
}

var bitConsts = []typ.Const{
	typ.C("Opt", int64(BitOpt)),
	typ.C("PK", int64(BitPK)),
	typ.C("Idx", int64(BitIdx)),
	typ.C("Uniq", int64(BitUniq)),
	typ.C("Ordr", int64(BitOrdr)),
	typ.C("Auto", int64(BitAuto)),
	typ.C("RO", int64(BitRO)),
}
