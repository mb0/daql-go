package main

import (
	"fmt"
	"log"
	"os"

	"xelf.org/daql/cmd"
	"xelf.org/daql/qry"
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
		log.Printf("no -data specified using empty project backend")
		bend = &qry.MemBackend{Reg: pr.Reg, Project: pr.Project}
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
