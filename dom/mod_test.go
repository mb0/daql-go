package dom

import (
	"testing"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
)

func TestMod(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"simple use", `(use 'auth') auth.dom.models/name`, "['Role' 'Acct' 'Cred' 'Sess']"},
		{"complex use", `(use 'blog') blog.Entry`, "<obj@blog.Entry>"},
		{"project use", `(use 'site') site.dom.schemas/name`, "['auth' 'blog']"},
	}
	reg := &lit.Reg{}
	files := mod.FileMods("testdata/")
	files.Ext = append(files.Ext, ".daql")
	files.Index = append(files.Index, "schema.daql")
	par := mod.NewLoaderEnv(extlib.Std, files)
	for _, test := range tests {
		prog := exp.NewProg(nil, reg, &Env{Par: par})
		res, err := prog.RunStr(test.raw, nil)
		if err != nil {
			t.Errorf("run %s got error: %+v", test.name, err)
			continue
		}
		got := res.String()
		if got != test.want {
			t.Errorf("res %s got %s want %s", test.name, got, test.want)
		}
	}
}

func TestPlainMod(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"mod init",
			`(use 'daql/dom' 'site') dom.projects/name`,
			"['site']",
		},
	}
	reg := &lit.Reg{}
	files := mod.FileMods("testdata/")
	files.Ext = append(files.Ext, ".daql")
	files.Index = append(files.Index, "schema.daql")
	par := mod.NewLoaderEnv(extlib.Std, mod.Registry, files)
	for _, test := range tests {
		prog := exp.NewProg(nil, reg, par)
		res, err := prog.RunStr(test.raw, nil)
		if err != nil {
			t.Errorf("run %s got error: %+v", test.name, err)
			continue
		}
		got := res.String()
		if got != test.want {
			t.Errorf("res %s got %s want %s", test.name, got, test.want)
		}
	}
}
