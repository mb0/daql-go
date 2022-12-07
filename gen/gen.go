package gen

import (
	"sort"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/xelf/bfr"
)

func Nogen(s *dom.Schema) bool {
	l, err := s.Extra.Key("nogen")
	return err == nil && !l.Nil()
}

// Gen is the code generation context holding the buffer and additional information.
type Gen struct {
	bfr.P
	*dom.Project
	Pkg    string
	Target string
	Header string
	OpPrec int
	Pkgs   map[string]string
	Imports
}

func (g *Gen) Prec(prec int) (restore func()) {
	org := g.OpPrec
	if org > prec {
		g.WriteByte('(')
	}
	g.OpPrec = prec
	return func() {
		if org > prec {
			g.WriteByte(')')
		}
		g.OpPrec = org
	}
}

// Prepend write each line in text prepended with prefix to the buffer.
// It strips the ascii whitespace bytes after the first linebreak, and tries to remove the same
// from each following line. If text starts with an empty line, that line is ignored.
func (g *Gen) Prepend(text, prefix string) {
	if text == "" {
		return
	}
	split := strings.Split(text, "\n")
	var ws int
	for i, s := range split {
		if ws == 0 && s == "" && len(split) > i+1 {
			goto Done
		}
		if ws == 0 {
			for len(s) > 0 {
				switch s[0] {
				case '\t', ' ':
					ws++
					s = s[1:]
				default:
					goto Done
				}
			}
		} else {
			for j := 0; j < ws && len(s) > 0; j++ {
				switch s[0] {
				case '\t', ' ':
					s = s[1:]
				default:
					goto Done
				}
			}
		}
	Done:
		g.Fmt("%s%s\n", prefix, s)
	}
}

// Imports has a list of alphabetically sorted dependencies. A dependency can be any string
// recognized by the generator. For go imports the dependency is a package path.
type Imports struct {
	List []string
}

// Add inserts path into the import list if not already present.
func (i *Imports) Add(path string) {
	idx := sort.SearchStrings(i.List, path)
	if idx < len(i.List) && i.List[idx] == path {
		return
	}
	i.List = append(i.List, "")
	copy(i.List[idx+1:], i.List[idx:])
	i.List[idx] = path
}
