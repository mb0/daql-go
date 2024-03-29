package gengo

import (
	"strings"
	"testing"

	"xelf.org/daql/dom"
	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
)

const barRaw = `(schema bar (Kind:enum X; Y; Z;))`
const fooRaw = `(schema foo
	(Align:bits A; B; C:3)
	(Kind:enum A; B; C;)
	(Node1; Name?:str)
	(Node2; Start:time)
	(Node3; Kind:<bits@bar.Kind>)
	(Node4; Kind:<bits@foo.Kind>)
	(Node5; @Node4)
)`

func TestWriteFile(t *testing.T) {
	_, err := dom.ReadSchema(nil, strings.NewReader(barRaw), "bar")
	if err != nil {
		t.Fatalf("schema bar error %v", err)
	}
	s, err := dom.ReadSchema(nil, strings.NewReader(fooRaw), "foo")
	if err != nil {
		t.Fatalf("schema foo error %v", err)
	}
	tests := []struct {
		model string
		want  string
	}{
		{"", "package foo\n"},
		{"align",
			"package foo\n\ntype Align uint32\n\n" +
				"const (\n" +
				"\tAlignA Align = 1 << iota\n" +
				"\tAlignB\n" +
				"\tAlignC = AlignA | AlignB\n" +
				")\n",
		},
		{"kind",
			"package foo\n\ntype Kind string\n\n" +
				"const (\n" +
				"\tKindA Kind = \"a\"\n" +
				"\tKindB Kind = \"b\"\n" +
				"\tKindC Kind = \"c\"\n" +
				")\n",
		},
		{"node1", "package foo\n\ntype Node1 struct {\n" +
			"\tName string `json:\"name,omitempty\"`\n" + "}\n",
		},
		{"node2", "package foo\n\nimport (\n\t\"time\"\n)\n\ntype Node2 struct {\n" +
			"\tStart time.Time `json:\"start\"`\n" + "}\n",
		},
		{"node3", "package foo\n\nimport (\n\t\"path/to/bar\"\n)\n\ntype Node3 struct {\n" +
			"\tKind bar.Kind `json:\"kind\"`\n" + "}\n",
		},
		{"node4", "package foo\n\ntype Node4 struct {\n" +
			"\tKind Kind `json:\"kind\"`\n" + "}\n",
		},
		{"node5", "package foo\n\ntype Node5 struct {\n" +
			"\tNode4\n" + "}\n",
		},
	}
	pkgs := map[string]string{
		"foo": "path/to/foo",
		"bar": "path/to/bar",
	}
	for _, test := range tests {
		var b strings.Builder
		c := &gen.Gen{P: bfr.P{Writer: &b}, Pkg: "path/to/foo", Pkgs: pkgs}
		ss := &dom.Schema{Name: s.Name}
		if m := s.Model(test.model); m != nil {
			ss.Models = []*dom.Model{m}
		}
		err := WriteSchema(c, ss)
		if err != nil {
			t.Errorf("write %s error: %v", test.model, err)
			continue
		}
		if got := b.String(); got != test.want {
			t.Errorf("for %s want %s got %s", test.model, test.want, got)
		}
	}
}
