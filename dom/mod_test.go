package dom

import (
	"testing"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/mod"
)

func TestMod(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"simple import", `(import 'auth') auth.dom.models/name`, "['Role' 'Acct' 'Cred' 'Sess']"},
		{"complex import", `(import 'blog') blog.Entry`, "<obj@blog.Entry>"},
		{"project import", `(import 'site') site.dom.schemas/name`, "['auth' 'blog']"},
	}
	par := mod.NewLoaderEnv(extlib.Std, mod.FileMods("testdata/"))
	for _, test := range tests {
		prog := exp.NewProg(&Env{Par: par})
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
			`(import 'daql/dom' 'site') dom.projects/name`,
			"['site']",
		},
	}
	par := mod.NewLoaderEnv(extlib.Std, mod.Registry, mod.FileMods("testdata/"))
	for _, test := range tests {
		prog := exp.NewProg(par)
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
