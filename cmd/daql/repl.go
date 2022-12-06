package main

import (
	"fmt"
	"os"

	xcmd "xelf.org/cmd"
	"xelf.org/daql/cmd"
	"xelf.org/daql/qry"
	"xelf.org/xelf/exp"
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
	if uri != "" {
		data, err := cmd.OpenData(pr, uri)
		if err != nil {
			return fmt.Errorf("open data: %v", err)
		}
		bend = data.Backend
	}
	q := qry.New(pr.Reg, xcmd.ProgRoot(), bend)
	r := xcmd.NewRepl(xcmd.ReplHistoryPath("daql/repl.history"))
	r.Wrap = func(env exp.Env) exp.Env {
		return &qry.Doc{Qry: q}
	}
	r.Run()
	return nil
}
