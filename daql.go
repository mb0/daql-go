package daql

import (
	"fmt"
	"path/filepath"
	"sort"

	"xelf.org/daql/dom"
	"xelf.org/daql/mig"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
)

type Project struct {
	Dir string
	Reg *lit.Regs
	mig.History
	mig.Record
}

func LoadProject(dir string) (*Project, error) {
	path, err := dom.DiscoverProject(dir)
	if err != nil {
		return nil, fmt.Errorf("discover project: %v", err)
	}
	reg := &lit.Regs{}
	h, err := mig.ReadHistory(reg, path)
	if err != nil && err != mig.ErrNoHistory {
		return nil, fmt.Errorf("read history: %v", err)
	}
	return &Project{filepath.Dir(path), reg, h, h.Curr()}, nil
}
func LoadProjectSchemas(dir string, args []string) (pr *Project, ss []*dom.Schema, err error) {
	pr, err = LoadProject(dir)
	if err != nil {
		return
	}
	ss = pr.Schemas
	if len(args) > 0 {
		ss, err = pr.FilterSchemas(args...)
		if err != nil {
			return
		}
	}
	return
}

func (pr *Project) FilterSchemas(names ...string) ([]*dom.Schema, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("requires list of schema names")
	}
	ss := make([]*dom.Schema, 0, len(names))
	for _, name := range names {
		s := pr.Schema(name)
		if s == nil {
			return nil, fmt.Errorf("schema %q not found", name)
		}
		ss = append(ss, s)
	}
	return ss, nil
}

func (pr *Project) Status(p *bfr.P) {
	cv := pr.First()
	last := pr.Last()
	lv := last.First()
	var vers string
	if !hasVers(lv) {
		vers = fmt.Sprintf("%s (unrecorded)", cv.Vers)
	} else if cv.Vers != lv.Vers || cv.Name != lv.Name {
		vers = fmt.Sprintf("%s (last recorded %s %s)", cv.Vers, lv.Vers,
			lv.Date.Format("2006-01-02 15:04"))
	} else {
		vers = fmt.Sprintf("%s (unchanged, recorded %s)",
			lv.Vers, lv.Date.Format("2006-01-02 15:04"))
	}
	p.Fmt("Project: %s %s\n", pr.Name, vers)
	changes := pr.Diff(pr.Last())
	chg(changes, cv.Name)
	for _, s := range pr.Schemas {
		v, _ := pr.Get(s.Qualified())
		p.Fmt("\n  %c %s %s\n", chg(changes, v.Name), v.Name, v.Vers)
		for _, m := range s.Models {
			v, _ = pr.Get(m.Qualified())
			p.Fmt("    %c %s %s\n", chg(changes, v.Name), v.Name, v.Vers)
		}
	}
	if lv != nil && chg(changes, lv.Name) != ' ' {
		p.Fmt("\nProject renamed from %s to %s\n", lv.Name, cv.Name)
	}
	if len(changes) > 0 {
		p.Fmt("\nDeletions:\n")
		dels := make([]string, 0, len(changes))
		for k := range changes {
			dels = append(dels, k)
		}
		sort.Strings(dels)
		for _, s := range dels {
			v, _ := pr.Get(s)
			p.Fmt("    - %s %s\n", s, v.Vers)
		}
	}
	p.Byte('\n')
}

func SchemaPath(pr *Project, s *dom.Schema) string {
	v, err := s.Extra.Key("file")
	if err == nil {
		loc := mod.ParseLoc(v.String())
		if loc.Proto() == "file" {
			return filepath.Dir(loc.Path())
		}
	}
	return pr.Dir
}

func chg(cm map[string]byte, name string) byte {
	if b, ok := cm[name]; ok {
		delete(cm, name)
		return b
	}
	return ' '
}

func hasVers(v *mig.Version) bool {
	if v == nil || v.Vers == "" {
		return false
	}
	vers, _ := mig.ParseVers(v.Vers)
	return vers != mig.Vers{}
}
