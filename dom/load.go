package dom

import (
	"fmt"
	"os"
	"path/filepath"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

type FileLoader struct {
	Proj  string
	Roots []string
}

func (l *FileLoader) Load(reg *lit.Reg, res string) (exp.Exp, string, error) {
	tries := []string{"schema.daql", fmt.Sprintf("%s.daql", filepath.Base(res))}
	for _, r := range l.Roots {
		path := filepath.Join(r, res)
		fi, err := os.Stat(path)
		if err != nil || !fi.IsDir() {
			continue
		}
		for _, try := range tries {
			p := filepath.Join(path, try)
			fi, err := os.Stat(p)
			if err != nil {
				continue
			}
			if fi.IsDir() {
				continue
			}
			x, rel, err := l.open(reg, p)
			if err != nil {
				return nil, "", fmt.Errorf("fileloader %s: %v", p, err)
			}
			return x, rel, nil
		}
	}
	return nil, "", fmt.Errorf("no schema found for path %q", res)
}

func (l *FileLoader) open(reg *lit.Reg, path string) (exp.Exp, string, error) {
	rel, err := filepath.Rel(filepath.Dir(l.Proj), path)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	x, err := exp.Read(reg, f, path)
	if err != nil {
		return nil, "", err
	}
	return x, rel, nil
}

var load = func() *loadSpec {
	n, _ := ext.NewNode(domReg, &Schema{})
	sig, _ := typ.Parse(`<form load str @>`)
	exp.SigRes(sig).Type = n.Type()
	return &loadSpec{SpecBase: exp.SpecBase{Decl: sig}}
}()

type loadSpec struct{ exp.SpecBase }

func (s *loadSpec) Value() lit.Val { return s }

func (s *loadSpec) Eval(p *exp.Prog, c *exp.Call) (*exp.Lit, error) {
	d := FindEnv(c.Env)
	if d == nil || d.Loader == nil {
		return nil, fmt.Errorf("no dom loader configured for %s", c)
	}
	a, err := p.Eval(c.Env, c.Args[0])
	if err != nil {
		return nil, err
	}
	rel, err := lit.ToStr(a.Val)
	if err != nil {
		return nil, err
	}
	x, path, err := d.Load(p.Reg, string(rel))
	if err != nil {
		return nil, err
	}
	x, err = p.Resl(c.Env, x, typ.Void)
	if err != nil {
		return nil, err
	}
	res, err := p.Eval(c.Env, x)
	if err != nil {
		return nil, err
	}
	sch := res.Val.(lit.Mut).Ptr().(*Schema)
	if sch.Extra == nil {
		sch.Extra = &lit.Dict{}
	}
	sch.Extra.SetKey("file", lit.Str(path))
	return res, nil
}
