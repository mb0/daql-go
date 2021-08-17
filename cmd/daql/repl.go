package main

import (
	"fmt"
	"log"
	"os"

	"xelf.org/daql/cmd"
	"xelf.org/daql/dom/domtest"
	"xelf.org/daql/qry"
	"xelf.org/xelf/lit"
)

func repl(args []string) error {
	pr, err := cmd.LoadProject(*dirFlag)
	if err != nil {
		return err
	}
	uri := *dataFlag
	if uri == "" {
		uri = os.Getenv("DAQL_DATA")
	}
	var bend qry.Backend
	if uri == "" {
		log.Printf("no -data specified using prod fixture")
		fix, err := domtest.ProdFixture(pr.Reg)
		if err != nil {
			return fmt.Errorf("parse fixture: %v", err)
		}
		membed := &qry.MemBackend{Reg: pr.Reg, Project: pr.Project}
		prodsch := fix.Schema("prod")
		for _, kv := range fix.Fix.Keyed {
			err = membed.Add(prodsch.Model(kv.Key), kv.Val.(*lit.List))
			if err != nil {
				return fmt.Errorf("prepare backend, add %s: %v", kv.Key, err)
			}
		}
		bend = membed
	} else {
		data, err := cmd.OpenData(pr, uri)
		if err != nil {
			return fmt.Errorf("open data: %v", err)
		}
		bend = data.Backend
	}
	r := cmd.NewRepl(pr.Reg, bend, cmd.ReplHistoryPath())
	r.Run()
	return nil
}
