package main

import (
	"fmt"

	"xelf.org/daql/cmd"
	"xelf.org/daql/dom/domtest"
	"xelf.org/daql/qry"
	"xelf.org/xelf/lit"
)

func repl(args []string) error {
	reg := &lit.Reg{}
	// use fixture and memory backend for now
	fix, err := domtest.ProdFixture()
	if err != nil {
		return fmt.Errorf("parse fixture: %v", err)
	}
	membed := &qry.MemBackend{Reg: reg, Project: &fix.Project}
	prodsch := fix.Schema("prod")
	for _, kv := range fix.Fix.Keyed {
		err = membed.Add(prodsch.Model(kv.Key), kv.Val.(*lit.List))
		if err != nil {
			return fmt.Errorf("prepare backend, add %s: %v", kv.Key, err)
		}
	}
	// TODO use the backup and a temporary database if we have a dataset argument
	// otherwise try the configured db
	r := cmd.NewRepl(reg, membed, cmd.ReplHistoryPath())
	r.Run()
	return nil
}
