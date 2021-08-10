package qry_test

import (
	"testing"

	. "xelf.org/daql/qry"

	"xelf.org/daql/dom/domtest"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

type MyQuery struct {
	Count int           `qry:"#prod.cat (gt .name 'd')"`
	All   []domtest.Cat `qry:"*prod.cat"`
	Top10 []domtest.Cat `qry:"*prod.cat lim:10"`
	Nest  *struct {
		domtest.Cat `qry:"+"`
		Prods       []struct {
			ID   int64
			Name string
		} `qry:"*prod.prod (eq .cat ..id) asc:name"`
	} `qry:"?prod.cat (eq .name $name)"`
}

func TestExecAuto(t *testing.T) {
	reg := &lit.Reg{}
	q := New(reg, extlib.Std, getBackend(reg))
	var res MyQuery
	mut, err := q.ExecAuto(&res, &lit.Dict{Keyed: []lit.KeyVal{
		{Key: "name", Val: lit.Str("a")},
	}})
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := `{count:3 all:[{id:25 name:'y'} {id:2 name:'b'} {id:3 name:'c'} {id:1 name:'a'} {id:4 name:'d'} {id:26 name:'z'} {id:24 name:'x'}] top10:[{id:25 name:'y'} {id:2 name:'b'} {id:3 name:'c'} {id:1 name:'a'} {id:4 name:'d'} {id:26 name:'z'} {id:24 name:'x'}] nest:{id:1 name:'a' prods:[{id:25 name:'Y'} {id:26 name:'Z'}]}}`
	got := mut.String()
	if got != want {
		t.Errorf("%T\ngot:  %s\nwant: %s", mut.Ptr(), got, want)
	} else {
		t.Logf("%s", got)
	}
}
func TestReflectQuery(t *testing.T) {
	reg := &lit.Reg{}
	var res MyQuery
	x, err := ReflectQuery(reg, &res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := `({} count:(#prod.cat (gt .name 'd')) all:(*prod.cat _ id; name;) top10:(*prod.cat lim:10 _ id; name;) nest:(?prod.cat (eq .name $name) + prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;)))`
	got := x.String()
	if got != want {
		t.Errorf("%T\ngot:  %s\nwant: %s", x, got, want)
	} else {
		t.Logf("%s", got)
	}
}
