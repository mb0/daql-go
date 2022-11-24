package qry

import (
	"testing"

	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/mod"
)

func TestPlainMod(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"mod init",
			`(use 'daql/qry') (project test)
			(qry.bend 'domtest:prod') ([] (#prod.cat) (#prod.prod))`,
			`[7 6]`,
		},
	}
	par := mod.NewLoaderEnv(extlib.Std, mod.Registry)
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
