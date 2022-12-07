package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	xcmd "xelf.org/cmd"
	_ "xelf.org/daql/dom"
	_ "xelf.org/daql/evt"
	"xelf.org/daql/gen"
	"xelf.org/daql/gen/gengo"
	"xelf.org/daql/mig"
	_ "xelf.org/daql/qry"
	"xelf.org/daql/xps/cmd"
	"xelf.org/xelf/bfr"
)

func Cmd(dir string, args []string) error {
	_, args = split(args)
	fst, args := split(args)
	switch fst {
	case "", "status":
		return status(dir, args)
	case "commit":
		return commit(dir, args)
	case "graph":
		return graph(dir, args)
	case "gen":
		return genGo(dir, args)
	case "repl":
		return cmd.Repl(dir, args)
	}
	return nil
}

func status(dir string, args []string) error {
	pr, err := cmd.LoadProject(dir)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(os.Stdout)
	pr.Status(&bfr.P{Writer: b})
	return b.Flush()
}

func commit(dir string, args []string) error {
	pr, err := cmd.LoadProject(dir)
	if err != nil {
		return err
	}
	err = pr.Commit(strings.Join(args, " "))
	if err == mig.ErrNoChanges {
		fmt.Printf("%s %s unchanged\n", pr.Name, pr.First().Vers)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s %s committed\n", pr.Name, pr.First().Vers)
	return nil
}

func graph(dir string, args []string) error {
	pr, ss, err := cmd.LoadProjectSchemas(dir, args)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(os.Stdout)
	err = cmd.GraphSchemas(&bfr.P{Writer: b}, pr, ss)
	if err != nil {
		return err
	}
	return b.Flush()
}

func genGo(dir string, args []string) error {
	pr, ss, err := cmd.LoadProjectSchemas(dir, args)
	if err != nil {
		return err
	}
	ppkg, err := xcmd.GoModPath(pr.Dir)
	if err != nil {
		return err
	}
	pkgs := gengo.DefaultPkgs()
	for _, s := range ss {
		if gen.Nogen(s) {
			continue
		}
		out := filepath.Join(cmd.SchemaPath(pr, s), fmt.Sprintf("%s_gen.go", s.Name))
		b := gengo.NewGenPkgs(pr.Project, s.Name, path.Join(ppkg, s.Name), pkgs)
		err := gengo.WriteSchemaFile(b, out, s)
		if err != nil {
			return err
		}
		fmt.Println(out)
	}
	return nil
}

func split(args []string) (string, []string) {
	if len(args) > 0 {
		return args[0], args[1:]
	}
	return "", nil
}
