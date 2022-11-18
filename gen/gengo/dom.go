package gengo

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"math/bits"
	"path/filepath"
	"sort"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

func WriteSchemaFile(g *gen.Gen, name string, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	g.P = bfr.P{Writer: b, Tab: "\t"}
	err := WriteSchema(g, s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(name, b.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("write schema file %s error: %v", name, err)
	}
	return nil
}

// WriteSchame writes the models with package and import declarations.
func WriteSchema(g *gen.Gen, s *dom.Schema) error {
	var embed string
	if s.Extra != nil {
		if _, err := s.Extra.Key("embed"); err == nil {
			if fn, err := s.Extra.Key("file"); err == nil {
				str, _ := lit.ToStr(fn)
				embed = filepath.Base(string(str))
			}
		}
	}
	b := bfr.Get()
	defer bfr.Put(b)
	// swap new buffer with context buffer
	tmp := g.Writer
	g.Writer = b
	for _, m := range s.Models {
		g.Byte('\n')
		err := WriteModel(g, m)
		if err != nil {
			return fmt.Errorf("write model %s: %w", m.Name, err)
		}
	}
	// swap back
	g.Writer = tmp
	g.Fmt("%spackage %s\n", g.Header, pkgName(g.Pkg))
	if len(g.Imports.List) > 0 {
		g.Fmt("\nimport (\n")
		if embed != "" {
			g.Fmt("\t_ \"embed\"\n\n")
		}
		groups := groupImports(g.Imports.List, "github")
		for i, gr := range groups {
			if i != 0 {
				g.Byte('\n')
			}
			for _, im := range gr {
				g.Fmt("\t\"%s\"\n", im)
			}
		}
		g.Fmt(")\n")
	} else if embed != "" {
		g.Fmt("import _ \"embed\"\n")
	}
	if embed != "" {
		g.Fmt("\n//go:embed %s\nvar rawSchema string\n", embed)
		g.Fmt("\nfunc RawSchema() string { return rawSchema }\n")
	}
	res, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("format %s: %w", b.Bytes(), err)
	}
	for len(res) > 0 {
		n, err := tmp.Write(res)
		if err != nil {
			return err
		}
		res = res[n:]
	}
	return nil
}

// WriteModel writes a type declaration for bits, enum and rec types.
// For bits and enum types the declaration includes the constant declarations.
func WriteModel(g *gen.Gen, m *dom.Model) (err error) {
	if doc, err := m.Extra.Key("doc"); err == nil {
		ch, err := lit.ToStr(doc)
		if ch != "" && err == nil {
			g.Prepend(fmt.Sprintf("%s %s", m.Name, ch), "// ")
		}
	}
	switch m.Kind.Kind {
	case knd.Bits:
		g.Fmt("type %s uint32\n\n", m.Name)
		writeBitsConsts(g, m)
	case knd.Enum:
		g.Fmt("type %s string\n\n", m.Name)
		writeEnumConsts(g, m)
	case knd.Obj:
		g.Fmt("type %s ", m.Name)
		err = writeRec(g, m.Type())
		g.Byte('\n')
	case knd.Func:
		ps := m.Params()
		last := len(ps) - 1
		if last > 0 {
			g.Fmt("type %sReq ", m.Name)
			err = WriteType(g, typ.Obj("", ps[:last]...))
			if err != nil {
				break
			}
			g.Fmt("\n\n")
		}
		g.Fmt("type %sRes ", m.Name)
		res := ps[last].Type
		err = WriteType(g, typ.Obj("",
			typ.P("Res?", res),
			typ.P("Err?", typ.Str),
		))
		if err != nil {
			break
		}
		g.Fmt("\n ")
		var tmp strings.Builder
		cc := *g
		cc.Writer = &tmp
		err = WriteType(&cc, res)
		if err != nil {
			break
		}
		g.Imports.Add("xelf.org/daql/hub")
		if last > 0 {
			g.Imports.Add("encoding/json")
			g.Fmt(`
type %[1]sFunc func(*hub.Msg, %[1]sReq) (%[2]s, error)

func (f %[1]sFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	var req %[1]sReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return nil, err
	}
	res, err := f(m, req)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}
`, m.Name, tmp.String())
		} else {
			g.Fmt(`
type %[1]sFunc func(*hub.Msg) (%[2]s, error)

func (f %[1]sFunc) Serve(m *hub.Msg) (*hub.Msg, error) {
	res, err := f(m)
	if err != nil {
		return nil, err
	}
	return m.ReplyRes(res), nil
}`, m.Name, tmp.String())
		}
	default:
		err = fmt.Errorf("model kind %s cannot be declared", m.Kind)
	}
	return err
}

func pkgName(pkg string) string {
	if idx := strings.LastIndexByte(pkg, '/'); idx != -1 {
		pkg = pkg[idx+1:]
	}
	if idx := strings.IndexByte(pkg, '.'); idx != -1 {
		pkg = pkg[:idx]
	}
	if idx := strings.IndexByte(pkg, 'n'); idx == 0 {
		pkg = pkg[idx+1:]
	}
	return pkg
}

func groupImports(list []string, pres ...string) (res [][]string) {
	other := make([]string, 0, len(list))
	rest := make([]string, 0, len(list))
Next:
	for _, im := range list {
		for _, pre := range pres {
			if strings.HasPrefix(im, pre) {
				rest = append(rest, im)
				continue Next
			}
		}
		other = append(other, im)
	}
	if len(other) > 0 {
		res = append(res, other)
	}
	if len(rest) > 0 {
		res = append(res, rest)
	}
	return res
}

func writeBitsConsts(g *gen.Gen, m *dom.Model) {
	mono := true
	g.Fmt("const (")
	for i, el := range m.Elems {
		g.Fmt("\n\t%s%s", m.Name, cor.Cased(el.Name))
		mask := uint64(el.Val)
		mono = mono && bits.OnesCount64(uint64(mask)) == 1
		if mono {
			if i == 0 {
				g.Fmt(" %s = 1 << iota", m.Name)
			}
		} else {
			g.Fmt(" = ")
			for j, cr := range constBits(m.Elems[:i], mask) {
				if j != 0 {
					g.Fmt(" | ")
				}
				g.Fmt("%s%s", m.Name, cor.Cased(cr.Name))
			}
		}
	}
	g.Fmt("\n)\n")
}

// Bits returns the matching constants s contained in mask. The given constants are checked in
// reverse and thus should match combined, more specific constants first.
func constBits(cs []*dom.Elem, mask uint64) []typ.Const {
	if len(cs) == 0 {
		return nil
	}
	res := make([]typ.Const, 0, 4)
	for i := len(cs) - 1; i >= 0 && mask != 0; i-- {
		e := cs[i]
		b := uint64(e.Val)
		if mask&b == b {
			mask &^= b
			res = append(res, typ.C(e.Name, e.Val))
		}
	}
	sort.Sort(consts(res))
	return res
}

type consts []typ.Const

func (cs consts) Len() int           { return len(cs) }
func (cs consts) Swap(i, j int)      { cs[i], cs[j] = cs[j], cs[i] }
func (cs consts) Less(i, j int) bool { return cs[i].Val < cs[j].Val }

func writeEnumConsts(g *gen.Gen, m *dom.Model) {
	g.Fmt("const (")
	for _, el := range m.Elems {
		g.Fmt("\n\t%s%s %s = \"%s\"",
			m.Name, cor.Cased(el.Name), m.Name, cor.Keyed(el.Name))
	}
	g.Fmt("\n)\n")
}
