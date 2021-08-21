package dom

import (
	"strings"
	"testing"

	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
)

func TestDom(t *testing.T) {
	tests := []struct {
		raw string
		str string
	}{
		{`(project app)`, `{name:'app' schemas:[]}`},
		{`(schema test)`, `{name:'test' models:[]}`},
		{`(project app (schema test))`, `{name:'app' schemas:[{name:'test' models:[]}]}`},
		{`(schema test label:'Test Schema')`,
			`{name:'test' extra:{label:'Test Schema'} models:[]}`,
		},
		{`(schema test (Dir:bits North; East; South; West;))`,
			`{name:'test' models:[{kind:<bits> name:'Dir' schema:'test' elems:[` +
				`{name:'North' val:1} {name:'East' val:2} ` +
				`{name:'South' val:4} {name:'West' val:8}]` +
				`}]}`,
		},
		{`(schema test (Dir:enum North; East; South; West;))`,
			`{name:'test' models:[{kind:<enum> name:'Dir' schema:'test' elems:[` +
				`{name:'North' val:1} {name:'East' val:2} ` +
				`{name:'South' val:3} {name:'West' val:4}]` +
				`}]}`,
		},
		{`(schema test (Named:obj prop:"something" Name:str))`,
			`{name:'test' models:[{kind:<obj> name:'Named' schema:'test' ` +
				`extra:{prop:'something'} ` +
				`elems:[{name:'Name' type:<str>}]` +
				`}]}`,
		},
		{`(schema test (Named:obj prop:true doc:"something" Name:str))`,
			`{name:'test' models:[{kind:<obj> name:'Named' schema:'test' ` +
				`extra:{prop:true doc:'something'} ` +
				`elems:[{name:'Name' type:<str>}]` +
				`}]}`,
		},
		{`(schema test (Point; X:int Y:int))`,
			`{name:'test' models:[{kind:<obj> name:'Point' schema:'test' ` +
				`elems:[{name:'X' type:<int>} {name:'Y' type:<int>}]}]}`,
		},
		{`(schema test (model Point obj (elem X int) (elem Y int)))`,
			`{name:'test' models:[{kind:<obj> name:'Point' schema:'test' ` +
				`elems:[{name:'X' type:<int>} {name:'Y' type:<int>}]}]}`,
		},
		{`(schema test (Named; (ID:uuid pk;) Name:str))`,
			`{name:'test' models:[{kind:<obj> name:'Named' schema:'test' elems:[` +
				`{name:'ID' type:<uuid> bits:2} {name:'Name' type:<str>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; B:str))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'B' type:<str>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; B:@test.Foo))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'B' type:<obj test.foo>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; @test.Foo;))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<obj test.foo>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; @test.Foo))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{type:<obj test.foo>}]}]}`,
		},
		{`(schema test (Foo:enum A;) (Bar; @test.Foo))`, `{name:'test' models:[` +
			`{kind:<enum> name:'Foo' schema:'test' elems:[{name:'A' val:1}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<enum test.foo>}]}]}`,
		},
		{`(schema test (Foo:enum A;) (Bar; @test.Foo;))`, `{name:'test' models:[` +
			`{kind:<enum> name:'Foo' schema:'test' elems:[{name:'A' val:1}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<enum test.foo>}]}]}`,
		},
		{`(schema test (Group; (ID:str pk;)) (Entry; (ID:int pk;) (Group:str ref:'..group')))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int> bits:2} ` +
				`{name:'Group' type:<str> ref:'..group'}]}]}`,
		},
		{`(schema test (Group; (ID:str pk;)) (Entry; (ID:int pk;) @Group.ID;)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int> bits:2} ` +
				`{name:'Group' type:<str> ref:'test.group'}]}]}`,
		},
		{`(schema test (Group; (ID:str pk;)) (Entry; (ID:int pk;) @Group.ID)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int> bits:2} ` +
				`{name:'Group' type:<str> ref:'test.group'}]}]}`,
		},
		{`(schema test (Group; (ID:str pk;)) (Entry; (ID:int pk;) Groups:list|@Group.ID)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int> bits:2} ` +
				`{name:'Groups' type:<list|str> ref:'test.group'}]}]}`,
		},
		{`(schema tree (Node; (ID:str pk;) Par:@.ID))`,
			`{name:'tree' models:[` +
				`{kind:<obj> name:'Node' schema:'tree' elems:[{name:'ID' type:<str> bits:2} ` +
				`{name:'Par' type:<str> ref:'tree.node'}]}]}`,
		},
		{`(schema test (Spam:func Egg:str bool))`, "{name:'test' models:[" +
			`{kind:<func> name:'Spam' schema:'test' elems:[{name:'Egg' type:<str>} {type:<bool>}]}]}`,
		},
	}
	for _, test := range tests {
		reg := &lit.Reg{}
		s, err := exp.Eval(reg, NewEnv(nil), test.raw)
		if err != nil {
			t.Errorf("execute %s got error: %+v", test.raw, err)
			continue
		}
		str := bfr.String(s)
		if str != test.str {
			t.Errorf("string equal want\n%s\n\tgot\n%s", test.str, str)
		}
		res, err := lit.Read(reg, strings.NewReader(test.str), "")
		if err != nil {
			t.Errorf("parse %s err: %+v", test.str, err)
		}
		got := bfr.String(res)
		if got != test.str {
			t.Errorf("read equal want\n%v\n\tgot\n%v", test.str, got)
		}
	}
}
