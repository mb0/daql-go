package qry

import (
	"fmt"
	"reflect"
	"strings"

	"xelf.org/xelf/bfr"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
)

// AutoQuery generates query from and saves the query result into a tagged go struct value pointer.
func AutoQuery(p *exp.Prog, pp interface{}, arg lit.Val) (lit.Mut, error) {
	x, err := ReflectQuery(pp)
	if err != nil {
		return nil, err
	}
	mut, err := lit.Proxy(p.Reg, pp)
	if err != nil {
		return nil, err
	}
	el, err := p.Run(x, arg)
	if err != nil {
		return nil, err
	}
	err = mut.Assign(el)
	if err != nil {
		return nil, err
	}
	return mut, nil
}

// ReflectQuery takes a tagged struct and generates and returns a query expression or an error.
// For now we just generate a query string which we then parse.
func ReflectQuery(pp interface{}) (exp.Exp, error) {
	pv := reflect.ValueOf(pp)
	if pv.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("requires pointer to struct got %T", pp)
	}
	pt := pv.Type().Elem()
	if pt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("requires pointer to struct got %T", pp)
	}
	buf := bfr.Get()
	defer bfr.Put(buf)
	buf.WriteString("({}\n")
	reflectStruct(pt, buf, 1)
	buf.WriteString("\n)")
	x, err := exp.Read(buf, "auto-qry")
	if err != nil {
		return nil, err
	}
	return x, nil
}

func reflectType(t reflect.Type, b bfr.Writer, depth int) {
	slice := t.Kind() == reflect.Slice
	if slice {
		t = t.Elem()
	}
	ptr := t.Kind() == reflect.Ptr
	if ptr {
		t = t.Elem()
	}
	switch k := t.Kind(); k {
	case reflect.Struct:
		b.WriteString(" ")
		reflectStruct(t, b, depth+1)
	}
}

func reflectStruct(t reflect.Type, b bfr.Writer, depth int) {
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		key := cor.Keyed(f.Name)
		if jtag := f.Tag.Get("json"); jtag != "" {
			if idx := strings.IndexByte(jtag, ','); idx >= 0 {
				key = jtag[:idx]
				if strings.Contains(jtag[idx:], ",omitempty") {
					key += "?"
				}
			} else {
				key = jtag
			}
			if key == "-" {
				continue
			}
		}
		qtag := f.Tag.Get("qry")
		if f.Anonymous {
			b.WriteString(qtag)
			continue
		}
		for tab := 0; tab < depth; tab++ {
			b.WriteByte('\t')
		}
		b.WriteString(key)
		if qtag == "" {
			b.WriteString(";")
		} else {
			b.WriteString(":(")
			b.WriteString(qtag)
			var tb strings.Builder
			reflectType(f.Type, &tb, 0)
			if got := tb.String(); len(got) > 1 {
				b.WriteByte(' ')
				switch got[1] {
				case '_', '-', '+':
				default:
					b.WriteByte('_')
				}
				b.WriteString(got)
			}
			b.WriteString(")")
		}
		if depth > 0 {
			b.WriteString("\n")
		}
	}
}
