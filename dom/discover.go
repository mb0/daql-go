package dom

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
)

const ProjectFileName = "project.xelf"

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

func OpenProject(reg *lit.Regs, path string) (*Project, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadProject(reg, f, path)
}

func ReadProject(reg *lit.Regs, r io.Reader, path string) (p *Project, _ error) {
	lit.UpdateRegs(reg, domReg)
	x, err := exp.Read(r, path)
	if err != nil {
		return nil, fmt.Errorf("read project %s: %v", path, err)
	}
	files := mod.FileMods(filepath.Dir(path))
	par := mod.NewLoaderEnv(extlib.Std, files)
	env := &Env{Par: par}
	v, err := exp.NewProg(env, reg).Run(x, nil)
	if err != nil {
		return nil, fmt.Errorf("eval project %s: %v", path, err)
	}
	p, ok := mutPtr(v).(*Project)
	if !ok {
		return nil, fmt.Errorf("expected *Project got %s", v.Value())
	}
	return p, nil
}

func OpenSchema(reg *lit.Regs, path string) (s *Schema, _ error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSchema(reg, f, path)
}

func ReadSchema(reg *lit.Regs, r io.Reader, path string) (s *Schema, _ error) {
	reg = lit.DefaultRegs(reg)
	lit.UpdateRegs(reg, domReg)
	x, err := exp.Read(r, path)
	if err != nil {
		return nil, err
	}
	v, err := exp.NewProg(NewEnv(), reg).Run(x, nil)
	if err != nil {
		return nil, err
	}
	s, ok := mutPtr(v).(*Schema)
	if !ok {
		return nil, fmt.Errorf("expected *Schema got %s", v.Value())
	}
	return s, nil
}
