package genpg

import (
	"fmt"
	"io"
	"os"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/typ"
)

func WriteSchemaFile(fname string, prog *exp.Prog, p *dom.Project, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	w := NewWriter(b, p, prog, nil)
	w.Project = p
	w.WriteString(w.Header)
	w.WriteString("BEGIN;\n\n")
	err := WriteSchema(w, s)
	if err != nil {
		return fmt.Errorf("render file %s error: %v", fname, err)
	}
	w.WriteString("COMMIT;\n")
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, b)
	return err
}

func WriteSchema(w *Writer, s *dom.Schema) (err error) {
	w.WriteString("CREATE SCHEMA ")
	w.WriteString(s.Name)
	w.WriteString(";\n\n")
	for _, m := range s.Models {
		switch m.Kind.Kind {
		case knd.Bits:
		case knd.Enum:
			err = w.WriteEnum(m)
		default:
			err = w.WriteTable(m)
		}
		if err != nil {
			return err
		}
		w.WriteString(";\n\n")
	}
	return nil
}

func (w *Writer) WriteEnum(m *dom.Model) error {
	w.WriteString("CREATE TYPE ")
	w.WriteString(m.Qualified())
	w.WriteString(" AS ENUM (")
	w.Indent()
	for i, c := range m.Consts() {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		WriteQuote(w, cor.Keyed(c.Name))
	}
	w.Dedent()
	return w.WriteByte(')')
}

func (w *Writer) WriteTable(m *dom.Model) error {
	w.WriteString("CREATE TABLE ")
	w.WriteString(m.Qualified())
	w.WriteString(" (")
	w.Indent()
	for i, p := range m.Params() {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		err := w.writeField(p, m.Elems[i])
		if err != nil {
			return err
		}
	}
	w.Dedent()
	return w.WriteByte(')')
}

func (w *Writer) writeField(p typ.Param, el *dom.Elem) error {
	key := p.Key
	if key == "" {
		switch p.Type.Kind & knd.Any {
		case knd.Bits, knd.Enum:
			split := strings.Split(typ.Name(p.Type), ".")
			key = split[len(split)-1]
		case knd.Obj:
			return w.writerEmbed(p.Type)
		default:
			return fmt.Errorf("unexpected embedded field type %s", p.Type)
		}
	}
	w.WriteString(key)
	w.WriteByte(' ')
	ts, err := TypString(p.Type)
	if err != nil {
		return err
	}
	if ts == "int8" && el.Bits&dom.BitPK != 0 && el.Bits&dom.BitAuto != 0 {
		w.WriteString("serial8")
	} else {
		w.WriteString(ts)
	}
	if el.Bits&dom.BitPK != 0 {
		w.WriteString(" PRIMARY KEY")
		// TODO auto
	} else if el.Bits&dom.BitOpt != 0 || p.Type.Kind&knd.None != 0 {
		w.WriteString(" NULL")
	} else {
		w.WriteString(" NOT NULL")
	}
	// TODO default
	// TODO references
	return nil
}

func (w *Writer) writerEmbed(t typ.Type) error {
	m := w.Project.Model(typ.Name(t))
	if m == nil {
		return fmt.Errorf("no model for %s", typ.Name(t))
	}
	for i, p := range m.Params() {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		if p.Key == "" {
			w.writerEmbed(p.Type)
			continue
		}
		w.WriteString(p.Key)
		w.WriteByte(' ')
		ts, err := TypString(p.Type)
		if err != nil {
			return err
		}
		w.WriteString(ts)
		if p.IsOpt() || p.Type.Kind&knd.None != 0 {
			w.WriteString(" NULL")
		} else {
			w.WriteString(" NOT NULL")
			// TODO implicit default
		}
	}
	return nil
}
