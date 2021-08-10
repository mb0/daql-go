package dom

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/lit"
)

const ProjectFileName = "project.daql"

// DiscoverProject looks for a project file based on path and returns a cleaned path.
//
// If path points to a file it check whether the file has a project file name. If path points to a
// directory, we try to look for a project file in the current and then in all its parents.
func DiscoverProject(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		if fi.Name() == ProjectFileName {
			return path, nil
		}
		path = filepath.Dir(path)
	}
	res, err := DiscoverProject(filepath.Join(path, ProjectFileName))
	if err == nil {
		return res, nil
	}
	dir := filepath.Dir(path)
	if dir == path {
		return "", err
	}
	return DiscoverProject(dir)
}

func OpenProject(path string) (*Project, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadProject(f, path)
}

func ReadProject(r io.Reader, path string) (p *Project, _ error) {
	reg := &lit.Reg{}
	reg.AddFrom(domReg)
	x, err := exp.Read(reg, r, path)
	if err != nil {
		return nil, fmt.Errorf("read project %s: %v", path, err)
	}
	env := NewEnv(nil)
	env.Loader = &FileLoader{Proj: path, Roots: []string{filepath.Dir(path)}}
	l, err := exp.EvalExp(reg, env, x)
	if err != nil {
		return nil, fmt.Errorf("eval project %s: %v", path, err)
	}
	mut, ok := l.Value().(lit.Mut)
	if ok {
		p, ok = mut.Ptr().(*Project)
	}
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", l.Value())
	}
	return p, nil
}

func OpenSchema(path string, pro *Project) (s *Schema, _ error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSchema(f, path, pro)
}

func ReadSchema(r io.Reader, path string, pro *Project) (s *Schema, _ error) {
	reg := &lit.Reg{}
	reg.AddFrom(domReg)
	if pro == nil {
		pro = &Project{}
	}
	n, err := ext.NewNode(reg, pro)
	if err != nil {
		return nil, err
	}
	x, err := exp.Read(reg, r, path)
	if err != nil {
		return nil, err
	}
	env := &ext.NodeEnv{Par: NewEnv(nil), Node: n}
	l, err := exp.EvalExp(reg, env, x)
	if err != nil {
		return nil, err
	}
	mut, ok := l.Value().(lit.Mut)
	if ok {
		s, ok = mut.Ptr().(*Schema)
	}
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", l.Value())
	}
	return s, nil
}
