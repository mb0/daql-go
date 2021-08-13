package evt_test

import (
	"strings"
	"testing"
	"time"

	"xelf.org/daql/dom"
	"xelf.org/daql/dom/domtest"
	"xelf.org/daql/evt"
	"xelf.org/daql/qry"
	"xelf.org/xelf/lit"
)

func TestLedger(t *testing.T) {
	l, err := testLedger()
	if err != nil {
		t.Fatalf("setup %v", err)
	}
	if !l.Rev().IsZero() {
		t.Fatalf("initial rev is not zero")
	}
	evs, err := l.Events(time.Time{})
	if err != nil {
		t.Fatalf("initial events %v", err)
	}
	if len(evs) != 0 {
		t.Fatalf("initial events not empty")
	}
	rev, evs, err := l.Publish(evt.Trans{Acts: []evt.Action{
		{evt.Sig{"prod.cat", "1"}, evt.CmdNew, map[string]lit.Val{
			"name": lit.Str("a"),
		}},
		{evt.Sig{"prod.prod", "25"}, evt.CmdNew, map[string]lit.Val{
			"name": lit.Str("Y"),
			"cat":  lit.Int(1),
		}},
	}})
	if err != nil {
		t.Fatalf("first %v", err)
	}
	if rev.IsZero() {
		t.Fatalf("pub rev is zero")
	}
	if !l.Rev().Equal(rev) {
		t.Fatalf("pub rev is not equal ledger rev")
	}
	evs, err = l.Events(time.Time{})
	if err != nil {
		t.Fatalf("pub events %v", err)
	}
	if len(evs) != 2 {
		t.Fatalf("pub events want 2 got %d", len(evs))
	}
	if id := evs[0].ID; id != 1 {
		t.Errorf("pub events want id 1 got %d", id)
	}
	if id := evs[1].ID; id != 2 {
		t.Errorf("pub events want id 2 got %d", id)
	}
	cats := l.Bend.Data["prod.cat"]
	if cats == nil || len(cats.Vals) != 1 {
		t.Errorf("pub cats %v", cats)
	}
	prods := l.Bend.Data["prod.prod"]
	if prods == nil || len(prods.Vals) != 1 {
		t.Errorf("pub prods %v", prods)
	}
}

func testLedger() (*evt.MemLedger, error) {
	reg := &lit.Reg{}
	p := &dom.Project{}
	ev, err := dom.OpenSchema(reg, "evt.daql", p)
	if err != nil {
		return nil, err
	}
	pr, err := dom.ReadSchema(reg, strings.NewReader(domtest.ProdRaw), "prod.daql", p)
	if err != nil {
		return nil, err
	}
	p.Schemas = append(p.Schemas, ev, pr)
	return evt.NewMemLedger(qry.NewMemBackend(&lit.Reg{}, p, nil))
}
