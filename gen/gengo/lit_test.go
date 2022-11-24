package gengo

import (
	"reflect"
	"strings"
	"testing"

	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib"
)

func TestWriteLit(t *testing.T) {
	tests := []struct {
		xelf    string
		want    string
		imports []string
	}{
		{`null`, "nil", nil},
		{`[]`, "[]lit.Val{}", []string{"xelf/lit"}},
		{`(make list|str)`, "[]string{}", nil},
		{`(time null)`, `*cor.Time("0001-01-01T00:00:00Z")`, []string{"xelf/cor"}},
	}
	for _, test := range tests {
		var b strings.Builder
		c := &gen.Gen{P: bfr.P{Writer: &b}, Pkgs: map[string]string{
			"lit": "xelf/lit",
			"cor": "xelf/cor",
		}}
		l, err := exp.NewProg(lib.Std).RunStr(test.xelf, nil)
		if err != nil {
			t.Errorf("parse %s err: %v", test.xelf, err)
			continue
		}
		err = WriteLit(c, l)
		if err != nil {
			t.Errorf("write %s error: %v", l, err)
			continue
		}
		res := b.String()
		if res != test.want {
			t.Errorf("want %s got %s", test.want, res)
		}
		if !reflect.DeepEqual(c.Imports.List, test.imports) {
			t.Errorf("test %s want imports %v got %v", test.xelf, test.imports, c.Imports)
		}
	}
}
