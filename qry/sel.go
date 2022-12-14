package qry

import (
	"errors"
	"fmt"
	"strings"

	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/typ"
)

// Sel describes the query selection of a subject type.
type Sel struct {
	Type typ.Type
	Fields
}

// Field represents a selected field that can either map to a subject field or an expression.
// Expression fields can also contain sub queries.
type Field struct {
	Key  string
	Name string
	Type typ.Type
	Exp  exp.Exp
	Sub  *Job
}

// Fields is a list of fields with a convenient lookup method.
type Fields []*Field

// Field returns the field with key or nil.
func (fs Fields) Field(key string) *Field {
	for _, f := range fs {
		if f.Key == key {
			return f
		}
	}
	return nil
}

func (fs Fields) with(n *Field) (_ Fields, found bool) {
	for i, f := range fs {
		if f.Key == n.Key {
			fs[i] = n
			return fs, true
		}
	}
	return append(fs, n), false
}

func (fs Fields) without(key string) (_ Fields, found bool) {
	for i, f := range fs {
		if f.Key == key {
			return append(fs[:i], fs[i+1:]...), true
		}
	}
	return fs, false
}

func paramField(p typ.Param) *Field {
	return &Field{Key: p.Key, Name: cor.Cased(p.Name), Type: p.Type}
}

func subjFields(t typ.Type) Fields {
	if k := t.Kind & knd.Obj; k == 0 || t.Kind&knd.All != k {
		return nil
	}
	pb := t.Body.(*typ.ParamBody)
	fs := make(Fields, 0, len(pb.Params))
	for _, p := range pb.Params {
		fs = append(fs, paramField(p))
	}
	return fs
}

func reslSel(p *exp.Prog, j *Job, ds []*exp.Tag) (*Sel, error) {
	if len(ds) == 0 {
		return &j.Subj.Sel, nil
	}
	fs := make(Fields, len(j.Subj.Fields))
	copy(fs, j.Subj.Fields)
	var mode byte
	for i, d := range ds {
		name := d.Tag
		switch name {
		case "+", "-":
			mode = name[0]
			if d.Exp != nil {
				return nil, fmt.Errorf("unexpected selection arguments %s", d)
			}
			continue
		case "_":
			mode = '+'
			if d.Exp == nil {
				fs = nil
				continue
			} else if len(ds[i:]) > 1 {
				return nil, fmt.Errorf("unexpected selection arguments %s", d)
			}
			el, err := p.Resl(j, simpleExpr(d.Exp), typ.Void)
			if err != nil && !errors.Is(err, exp.ErrDefer) {
				return nil, err
			}
			et := typ.Res(el.Type())
			f := &Field{Key: cor.Keyed(name), Name: cor.Cased(name), Exp: el, Type: et}
			return &Sel{Type: f.Type, Fields: Fields{f}}, nil
		case "":
			return nil, fmt.Errorf("unnamed selection %s", d)
		}
		switch name[0] {
		case '-', '+':
			mode = name[0]
			name = name[1:]
		default:
		}
		key := strings.ToLower(name)
		switch mode {
		case '-': // exclude
			if d.Exp != nil {
				return nil, fmt.Errorf("unexpected selection arguments %s", d)
			}
			fs, _ = fs.without(key)
		case '+':
			if d.Exp == nil { // naked selects choose a subj field by key
				p := findParam(j.Subj.Type, key)
				if p == nil {
					return nil, fmt.Errorf("no param for key %s", key)
				}
				fs, _ = fs.with(paramField(*p))
			} else {
				name := cor.Cased(name)
				el, err := p.Resl(j, d.Exp, typ.Void)
				if err != nil {
					return nil, err
				}
				d.Exp = el
				f := &Field{Key: key, Name: name, Exp: el}
				call, ok := el.(*exp.Call)
				if ok {
					if s := exp.UnwrapSpec(call.Spec); s != nil {
						if sub, ok := call.Env.(*Job); ok && sub != j {
							f.Sub = sub
							f.Type = sub.Res
						}
					}
				}
				if f.Type == typ.Void {
					f.Type = typ.Res(el.Type())
				}
				switch k := f.Type.Kind; k & knd.Data {
				case knd.Num:
					f.Type = typ.Real
					f.Type.Kind |= (k & knd.None)
				case knd.Char:
					f.Type = typ.Str
					f.Type.Kind |= (k & knd.None)
				}
				fs, _ = fs.with(f)
			}
		}
	}
	ps := make([]typ.Param, 0, len(fs))
	for _, f := range fs {
		ps = append(ps, typ.P(f.Name, f.Type))
	}
	return &Sel{Type: typ.Obj("", ps...), Fields: fs}, nil
}

func simpleExpr(el exp.Exp) exp.Exp {
	s, ok := el.(*exp.Sym)
	if ok && cor.IsKey(s.Sym) {
		s := *s
		s.Sym = "." + s.Sym
		return &s
	}
	return el
}

func findParam(t typ.Type, key string) *typ.Param {
	b, ok := t.Body.(*typ.ParamBody)
	if !ok {
		return nil
	}
	i := b.FindKeyIndex(key)
	if i < 0 {
		return nil
	}
	return &b.Params[i]
}
