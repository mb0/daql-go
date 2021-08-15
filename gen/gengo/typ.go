package gengo

import (
	"fmt"
	"strings"

	"xelf.org/daql/gen"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/typ"
)

// WriteType writes the native go type for t to c or returns an error.
func WriteType(g *gen.Gen, t typ.Type) error {
	switch t.Kind {
	case knd.Any:
		return g.Fmt(Import(g, "lit.Val"))
	case knd.Tupl:
		return g.Fmt(Import(g, "*exp.Tupl"))
	case knd.Typ:
		return g.Fmt(Import(g, "typ.Type"))
	case knd.Exp:
		return g.Fmt(Import(g, "exp.Exp"))
	}
	var r string
	switch t.Kind & knd.Data {
	case knd.Bool:
		r = "bool"
	case knd.Int:
		r = "int64"
	case knd.Real:
		r = "float64"
	case knd.Str:
		r = "string"
	case knd.Raw:
		r = "[]byte"
	case knd.UUID:
		r = "[16]byte"
	case knd.Time:
		r = Import(g, "time.Time")
	case knd.Span:
		r = Import(g, "time.Duration")
	case knd.List:
		g.Fmt("[]")
		return WriteType(g, typ.ContEl(t))
	case knd.Dict:
		if el := typ.ContEl(t); el != typ.Any {
			g.Fmt("map[string]")
			return WriteType(g, el)
		}
		return g.Fmt(Import(g, "*lit.Dict"))
	case knd.Rec:
		opt := t.Kind&knd.None != 0
		if opt {
			g.Byte('*')
		}
		g.Fmt("struct {\n")
		b, ok := t.Body.(*typ.ParamBody)
		if !ok || len(b.Params) == 0 {
			return fmt.Errorf("invalid")
		}
		for _, f := range b.Params {
			opt := f.IsOpt()
			g.Byte('\t')
			if f.Name != "" {
				g.Fmt(cor.Cased(f.Name))
				g.Byte(' ')
			}
			err := WriteType(g, f.Type)
			if err != nil {
				return fmt.Errorf("write field %s: %w", f.Name, err)
			}
			if f.Key != "" {
				g.Fmt(" `json:\"")
				g.Fmt(f.Key)
				if opt {
					g.Fmt(",omitempty")
				}
				g.Fmt("\"`")
			}
			g.Byte('\n')
		}
		return g.Byte('}')
	case knd.Bits, knd.Enum, knd.Obj:
		name := typ.Name(t)
		if i := strings.LastIndexByte(name, '.'); i >= 0 {
			name = name[:i+1] + cor.Cased(name[i+1:])
		}
		r = Import(g, name)
	}
	if r == "" {
		return fmt.Errorf("type %s cannot be represented in go", t)
	}
	if t.Kind&knd.None != 0 {
		g.Byte('*')
	}
	return g.Fmt(r)
}
