package dom

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

type FileLoader struct {
	Proj  string
	Roots []string
}

func (l *FileLoader) Load(reg *lit.Reg, res string) (exp.Exp, string, error) {
	var tries []string
	if strings.HasSuffix(res, ".daql") {
		tries = append(tries, res)
	} else {
		f := res + ".daql"
		tries = []string{f, res + "/schema.daql", res + "/" + f}
	}
	for _, r := range l.Roots {
		for _, try := range tries {
			x, rel, err := l.try(filepath.Join(r, try))
			if err != nil {
				if err == errNotFound {
					continue
				}
				return nil, "", err
			}
			return x, rel, nil
		}
	}
	return nil, "", fmt.Errorf("no schema found for %q", res)
}

var errNotFound = errors.New("not found")

func (l *FileLoader) try(path string) (exp.Exp, string, error) {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return nil, "", errNotFound
	}
	x, rel, err := l.open(path)
	if err != nil {
		return nil, "", fmt.Errorf("fileloader %s: %w", path, err)
	}
	return x, rel, nil
}

func (l *FileLoader) open(path string) (exp.Exp, string, error) {
	rel, err := filepath.Rel(filepath.Dir(l.Proj), path)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	x, err := exp.Read(f, path)
	if err != nil {
		return nil, "", err
	}
	return x, rel, nil
}

var load = func() *loadSpec {
	sig, _ := typ.Parse(`<form@load str @>`)
	exp.SigRes(sig).Type = schemaSpec.Type()
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
