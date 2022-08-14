package gengo

import (
	"reflect"
	"strings"
	"testing"

	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/typ"
)

func TestWriteType(t *testing.T) {
	tests := []struct {
		t       typ.Type
		s       string
		imports []string
	}{
		{typ.Any, "lit.Val", []string{"xelf/lit"}},
		{typ.ListOf(typ.Any), "[]lit.Val", []string{"xelf/lit"}},
		{typ.ListOf(typ.Time), "[]time.Time", []string{"time"}},
		{typ.DictOf(typ.Any), "*lit.Dict", []string{"xelf/lit"}},
		{typ.DictOf(typ.Time), "map[string]time.Time", []string{"time"}},
		{typ.Bool, "bool", nil},
		{typ.Span, "time.Duration", []string{"time"}},
		{typ.Obj("",
			typ.P("Foo", typ.Str),
			typ.P("Bar?", typ.Int),
			typ.P("Spam", typ.Opt(typ.Int)),
		), "struct {\n\t" +
			"Foo string `json:\"foo\"`\n\t" +
			"Bar int64 `json:\"bar,omitempty\"`\n\t" +
			"Spam *int64 `json:\"spam\"`\n}", nil},
	}
	for _, test := range tests {
		var b strings.Builder
		c := &gen.Gen{P: bfr.P{Writer: &b}, Pkgs: map[string]string{
			"lit": "xelf/lit",
		}}
		err := WriteType(c, test.t)
		if err != nil {
			t.Errorf("test %s error: %v", test.s, err)
			continue
		}
		res := b.String()
		if res != test.s {
			t.Errorf("test %s got %s", test.s, res)
		}
		if !reflect.DeepEqual(c.Imports.List, test.imports) {
			t.Errorf("test %s want imports %v got %v", test.s, test.imports, c.Imports)
		}
	}
}
