package gengo

import (
	"fmt"

	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// WriteLit writes the native go literal for l to c or returns an error.
func WriteLit(g *gen.Gen, l *exp.Lit) error { return WriteVal(g, l.Res, l.Val) }

// WriteLit writes the native go literal for l to c or returns an error.
func WriteVal(g *gen.Gen, t typ.Type, v lit.Val) error {
	if t == typ.Void {
		t = v.Type()
	}
	if v.Nil() {
		return g.Fmt("nil")
	}
	opt := t.Kind&knd.None != 0
	switch k := t.Kind; k & knd.Any {
	case knd.Num, knd.Bool, knd.Int, knd.Real:
		if opt {
			call := "cor.Real"
			switch k & knd.Data {
			case knd.Bool:
				call = "cor.Bool"
			case knd.Int:
				call = "cor.Int"
			}
			return writeCall(g, call, v)
		} else {
			g.WriteString(v.String())
		}
	case knd.Char, knd.Str:
		if opt {
			return writeCall(g, "cor.Str", v)
		} else {
			return v.Print(&bfr.P{Writer: g.Writer, JSON: true})
		}
	case knd.Raw:
		if !opt {
			g.Byte('*')
		}
		return writeCall(g, "cor.Raw", v)
	case knd.UUID:
		if !opt {
			g.Byte('*')
		}
		return writeCall(g, "cor.UUID", v)
	case knd.Time:
		if !opt {
			g.Byte('*')
		}
		return writeCall(g, "cor.Time", v)
	case knd.Span:
		if !opt {
			g.Byte('*')
		}
		return writeCall(g, "cor.Span", v)
	case knd.List:
		g.Fmt("[]")
		err := WriteType(g, typ.ContEl(t))
		if err != nil {
			return err
		}
		return writeIdxer(g, v)
	case knd.Dict:
		g.Fmt("map[string]")
		err := WriteType(g, typ.ContEl(t))
		if err != nil {
			return err
		}
		return writeKeyer(g, v, func(i int, k string, e lit.Val) error {
			g.Fmt("%q: ", k)
			return WriteVal(g, e.Type(), e)
		})
	case knd.Rec, knd.Obj:
		if opt {
			g.Byte('&')
		}
		err := WriteType(g, typ.Deopt(t))
		if err != nil {
			return err
		}
		return writeKeyer(g, v, func(i int, k string, e lit.Val) error {
			g.Fmt("%s: ", k)
			return WriteVal(g, e.Type(), e)
		})
	case knd.Bits, knd.Enum:
	}
	return nil
}

func writeCall(g *gen.Gen, name string, v lit.Val) error {
	g.Fmt(Import(g, name))
	g.Byte('(')
	err := v.Print(&bfr.P{Writer: g.Writer, JSON: true})
	g.Byte(')')
	return err
}

func writeIdxer(g *gen.Gen, vv lit.Val) error {
	v, ok := vv.(lit.Idxr)
	if !ok {
		return fmt.Errorf("expect idxer got %T", vv)
	}
	g.Byte('{')
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			g.Fmt(", ")
		}
		e, err := v.Idx(i)
		if err != nil {
			return err
		}
		err = WriteVal(g, e.Type(), e)
		if err != nil {
			return err
		}
	}
	return g.Byte('}')
}

func writeKeyer(g *gen.Gen, vv lit.Val, el func(int, string, lit.Val) error) error {
	v, ok := vv.(lit.Keyr)
	if !ok {
		return fmt.Errorf("expect keyer got %T", vv)
	}
	g.Byte('{')
	keys := v.Keys()
	for i, k := range keys {
		if i > 0 {
			g.Fmt(", ")
		}
		e, err := v.Key(k)
		if err != nil {
			return err
		}
		err = el(i, k, e)
		if err != nil {
			return err
		}
	}
	return g.Byte('}')
}
