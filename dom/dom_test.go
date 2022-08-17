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
				`{name:'ID' type:<uuid@test.Named.ID> bits:2} {name:'Name' type:<str>}]}]}`,
		},
		{`(schema test (Named; ID:uuid Name:str))`,
			`{name:'test' models:[{kind:<obj> name:'Named' schema:'test' elems:[` +
				`{name:'ID' type:<uuid@test.Named.ID> bits:2} {name:'Name' type:<str>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; B:str))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'B' type:<str>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; B:@Foo))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'B' type:<obj@test.Foo>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; @Foo;))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<obj@test.Foo>}]}]}`,
		},
		{`(schema test (Foo; A:str) (Bar; @Foo))`, `{name:'test' models:[` +
			`{kind:<obj> name:'Foo' schema:'test' elems:[{name:'A' type:<str>}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{type:<obj@test.Foo>}]}]}`,
		},
		{`(schema test (Foo:enum A;) (Bar; @Foo))`, `{name:'test' models:[` +
			`{kind:<enum> name:'Foo' schema:'test' elems:[{name:'A' val:1}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<enum@test.Foo>}]}]}`,
		},
		{`(schema test (Foo:enum A;) (Bar; @Foo;))`, `{name:'test' models:[` +
			`{kind:<enum> name:'Foo' schema:'test' elems:[{name:'A' val:1}]} ` +
			`{kind:<obj> name:'Bar' schema:'test' elems:[{name:'Foo' type:<enum@test.Foo>}]}]}`,
		},
		{`(schema test (Group; ID:str) (Entry; ID:int @Group.ID;)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str@test.Group.ID> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int@test.Entry.ID> bits:2} ` +
				`{name:'Group' type:<str@test.Group.ID>}]}]}`,
		},
		{`(schema test (Group; ID:str) (Entry; ID:int @Group.ID)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str@test.Group.ID> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int@test.Entry.ID> bits:2} ` +
				`{name:'Group' type:<str@test.Group.ID>}]}]}`,
		},
		{`(schema test (Group; ID:str) (Entry; ID:int Groups:list|@Group.ID)))`,
			`{name:'test' models:[` +
				`{kind:<obj> name:'Group' schema:'test' elems:[{name:'ID' type:<str@test.Group.ID> bits:2}]} ` +
				`{kind:<obj> name:'Entry' schema:'test' elems:[` +
				`{name:'ID' type:<int@test.Entry.ID> bits:2} ` +
				`{name:'Groups' type:<list|str@test.Group.ID>}]}]}`,
		},
		{`(schema tree (Node; ID:str Par:@.ID))`,
			`{name:'tree' models:[` +
				`{kind:<obj> name:'Node' schema:'tree' elems:[{name:'ID' type:<str@tree.Node.ID> bits:2} ` +
				`{name:'Par' type:<str@tree.Node.ID>}]}]}`,
		},
		{`(schema tree (Node; ID:str@@ Par:@Node.ID))`,
			`{name:'tree' models:[` +
				`{kind:<obj> name:'Node' schema:'tree' elems:[{name:'ID' type:<str@tree.Node.ID> bits:2} ` +
				`{name:'Par' type:<str@tree.Node.ID>}]}]}`,
		},
		{`(schema test (Spam:func Egg:str bool))`, "{name:'test' models:[" +
			`{kind:<func> name:'Spam' schema:'test' elems:[{name:'Egg' type:<str>} {type:<bool>}]}]}`,
		},
		{`(schema test (Node; (Egg:str idx;)))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Egg' type:<str> bits:4}]}]}`,
		},
		{`(schema test (Node; (Egg:str uniq;)))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Egg' type:<str> bits:8}]}]}`,
		},
		{`(schema test (Node; Spam:str Egg:str idx:['spam' 'egg']))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Spam' type:<str>} {name:'Egg' type:<str>}] ` +
			`object:{indices:[{keys:['spam' 'egg']}]}}]}`,
		},
		{`(schema test (Node; Spam:str Egg:str idx:'spam' idx:'egg'))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Spam' type:<str>} {name:'Egg' type:<str>}] ` +
			`object:{indices:[{keys:['spam']} {keys:['egg']}]}}]}`,
		},
		{`(schema test (Node; Spam:str Egg:str uniq:'spam' uniq:'egg'))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Spam' type:<str>} {name:'Egg' type:<str>}] ` +
			`object:{indices:[{keys:['spam'] unique:true} {keys:['egg'] unique:true}]}}]}`,
		},
		{`(schema test (Node; Spam:str Egg:str uniq:['spam' 'egg']))`, "{name:'test' models:[" +
			`{kind:<obj> name:'Node' schema:'test' elems:[{name:'Spam' type:<str>} {name:'Egg' type:<str>}] ` +
			`object:{indices:[{keys:['spam' 'egg'] unique:true}]}}]}`,
		},
	}
	for _, test := range tests {
		reg := &lit.Reg{}
		s, err := exp.Eval(nil, reg, NewEnv(nil), test.raw)
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
