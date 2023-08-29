package qry_test

import (
	"log"
	"testing"
	"time"

	. "xelf.org/daql/qry"

	"xelf.org/daql/dom/domtest"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

func getBackend(reg *lit.Regs) Backend {
	f := domtest.Must(domtest.ProdFixture(reg))
	b := NewMemBackend(&f.Project, f.Version)
	s := f.Schema("prod")
	for _, kv := range f.Fix {
		err := b.Add(s.Model(kv.Key), kv.Val.(*lit.Vals))
		if err != nil {
			log.Printf("test backend error: %v", err)
		}
	}
	return b
}

// TestQryMutSel makes shure that mut expressions in query selections work without side effects.
func TestQryMutSel(t *testing.T) {
	reg := lit.NewRegs()
	b := getBackend(reg)
	tests := []struct {
		Raw  string
		Want string
	}{
		{`(?prod.cat _:(.name + '!'))`, `'y!'`},
		{`(?prod.cat _:(.name + '!'))`, `'y!'`},
		{`(?dom.model _:(.schema + '.' .name))`, `'prod.Cat'`},
		{`(?dom.model _:(.schema + '.' .name))`, `'prod.Cat'`},
	}
	for _, test := range tests {
		el, err := exp.NewProg(NewDoc(extlib.Std, b)).RunStr(test.Raw, nil)
		if err != nil {
			t.Errorf("qry %s failed: %v", test.Raw, err)
			continue
		}
		if el == nil {
			t.Errorf("qry %s got nil result", test.Raw)
			continue
		}
		if got := bfr.String(el); got != test.Want {
			t.Errorf("want for %s\n\t%s got %s", test.Raw, test.Want, got)
			continue
		}
	}
}

func TestQry(t *testing.T) {
	reg := lit.NewRegs()
	b := getBackend(reg)
	tests := []struct {
		Raw  string
		Want string
	}{
		{`(#prod.cat)`, `7`},
		{`(#prod.prod)`, `6`},
		{`([]+ (#prod.cat) (#prod.prod))`, `[7 6]`},
		{`({} cats:(#prod.cat) prods:(#prod.prod))`, `{cats:7 prods:6}`},
		{`(#prod.cat off:5 lim:5)`, `2`},
		{`(#prod.prod (eq .cat $int1))`, `2`},
		{`(?prod.cat)`, `{id:25 name:'y'}`},
		{`(?$list)`, `'a'`},
		{`(#$list)`, `3`},
		{`(#prod.cat (gt .name 'd'))`, `3`},
		{`(?prod.cat (eq .id 1) _:name)`, `'a'`},
		{`(?prod.cat (eq .id $int1) _:name)`, `'a'`},
		{`(?prod.cat (eq .name 'a'))`, `{id:1 name:'a'}`},
		{`(?prod.cat (eq .name $strA))`, `{id:1 name:'a'}`},
		{`(?prod.cat _ id;)`, `{id:25}`},
		{`(?prod.cat _:id)`, `25`},
		{`(?prod.cat off:1)`, `{id:2 name:'b'}`},
		{`(?prod.cat off:$int1)`, `{id:2 name:'b'}`},
		{`(*prod.cat lim:2)`, `[{id:25 name:'y'} {id:2 name:'b'}]`},
		{`(*prod.cat asc:name off:1 lim:2)`, `[{id:2 name:'b'} {id:3 name:'c'}]`},
		{`(*prod.cat desc:name lim:2)`, `[{id:26 name:'z'} {id:25 name:'y'}]`},
		{`(?prod.label _ id; label:('Label: '+ .name))`, `{id:1 label:'Label: M'}`},
		{`(*prod.label off:1 lim:2 - tmpl;)`, `[{id:2 name:'N'} {id:3 name:'O'}]`},
		{`(*prod.prod desc:cat asc:name lim:3)`,
			`[{id:1 name:'A' cat:3} {id:3 name:'C' cat:3} {id:2 name:'B' cat:2}]`},
		{`(?prod.cat (eq .name 'c') +
			prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;)
		)`, `{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}`},
		{`(*prod.cat (or (eq .name 'b') (eq .name 'c')) asc:name +
			prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;)
		)`, `[{id:2 name:'b' prods:[{id:2 name:'B'} {id:4 name:'D'}]} ` +
			`{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}]`},
		{`(*prod.prod (ge .id 25) + catn:(?prod.cat (eq .id ..cat) _:name))`,
			`[{id:25 name:'Y' cat:1 catn:'a'} {id:26 name:'Z' cat:1 catn:'a'}]`},
		{`(*dom.model (eq .schema 'prod') _:name)`, `['Cat' 'Prod' 'Label']`},
		{`(?dom.model (eq .name 'Cat'))`, `{kind:<obj> name:'Cat' schema:'prod' extra:{topic:true} elems:[{name:'ID' type:<int@prod.Cat.ID> bits:2} {name:'Name' type:<str>}] object;}`},
	}
	param := lit.MakeObj(lit.Keyed{
		{Key: "int1", Val: lit.Int(1)},
		{Key: "strA", Val: lit.Str("a")},
		{Key: "list", Val: lit.NewList(typ.Str,
			lit.Str("a"),
			lit.Str("b"),
			lit.Str("c"),
		)},
	})
	for _, test := range tests {
		start := time.Now()
		el, err := exp.NewProg(NewDoc(extlib.Std, b)).RunStr(test.Raw, param)
		end := time.Now()
		if err != nil {
			t.Errorf("qry %s failed: %v", test.Raw, err)
			continue
		}
		if el == nil {
			t.Errorf("qry %s got nil result", test.Raw)
			continue
		}
		if got := bfr.String(el); got != test.Want {
			t.Errorf("want for %s\n\t%s got %s", test.Raw, test.Want, got)
			continue
		}
		log.Printf("test took %s", end.Sub(start))
	}
}

func TestQryType(t *testing.T) {
	reg := lit.NewRegs()
	b := getBackend(reg)
	tests := []struct {
		Raw  string
		Want string
	}{
		{`(#prod.cat)`, `<int>`},
		{`([]+ (#prod.cat) (#prod.prod))`, `<idxr>`},
		{`({} cats:(#prod.cat) prods:(#prod.prod))`, `<keyr>`},
		{`(list|int + (#prod.cat) (#prod.prod))`, `<list|int>`},
		{`(dict|int cats:(#prod.cat) prods:(#prod.prod))`, `<dict|int>`},
		{`(?prod.cat)`, `<obj@prod.Cat?>`},
		{`(?$list)`, `<str?>`},
		{`(?prod.cat _ id;)`, `<obj? ID:int@prod.Cat.ID>`},
		{`(*prod.cat _ id)`, `<list|obj ID:int@prod.Cat.ID>`},
		{`(?prod.cat _:id)`, `<int@prod.Cat.ID?>`},
		{`(*prod.cat _:id)`, `<list|int@prod.Cat.ID>`},
		{`(*prod.cat lim:2)`, `<list|obj@prod.Cat>`},
		{`(?prod.label _ id; label:('Label: '+ .name))`,
			`<obj? ID:int@prod.Label.ID Label:str>`},
	}
	param := lit.MakeObj(lit.Keyed{
		{Key: "int1", Val: lit.Int(1)},
		{Key: "strA", Val: lit.Str("a")},
		{Key: "list", Val: lit.NewList(typ.Str,
			lit.Str("a"),
			lit.Str("b"),
			lit.Str("c"),
		)},
	})
	for _, test := range tests {
		el, err := exp.NewProg(NewDoc(extlib.Std, b)).RunStr(test.Raw, param)
		if err != nil {
			t.Errorf("qry %s failed: %v", test.Raw, err)
			continue
		}
		if el == nil {
			t.Errorf("qry %s got nil result", test.Raw)
			continue
		}
		if got := bfr.String(el.Type()); got != test.Want {
			t.Errorf("want for %s\n\t%s got %s", test.Raw, test.Want, got)
			continue
		}
	}
}
