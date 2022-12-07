package cmd

import (
	"fmt"

	"xelf.org/cmd"
	"xelf.org/daql/qry"
	"xelf.org/xelf/exp"
)

func Repl(dir string, args []string) error {
	pr, err := LoadProject(dir)
	if err != nil {
		return err
	}
	var bend qry.Backend
	if len(args) > 0 {
		data, err := qry.Open(pr.Project, args[0])
		if err != nil {
			return fmt.Errorf("open data: %v", err)
		}
		bend = data.Backend
	}
	q := qry.New(pr.Reg, cmd.ProgRoot(), bend)
	r := cmd.NewRepl(cmd.ReplHistoryPath("daql/repl.history"))
	r.Wrap = func(env exp.Env) exp.Env {
		return &qry.Doc{Qry: q}
	}
	r.Run()
	return nil
}
