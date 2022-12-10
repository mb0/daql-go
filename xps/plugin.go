package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"xelf.org/daql"
	_ "xelf.org/daql/evt"
	"xelf.org/daql/gen"
	"xelf.org/daql/gen/gengo"
	"xelf.org/daql/mig"
	"xelf.org/daql/qry"
	"xelf.org/daql/xps/prov"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/xps"
)

func Cmd(ctx *xps.CmdCtx) error {
	switch ctx.Split() {
	case "", "status":
		return status(ctx)
	case "commit":
		return commit(ctx)
	case "graph":
		return graph(ctx)
	case "gen":
		return genGo(ctx)
	case "repl":
		return repl(ctx)
	}
	return nil
}

func repl(ctx *xps.CmdCtx) error {
	pr, err := daql.LoadProject(ctx.Dir)
	if err != nil {
		return err
	}
	var bend qry.Backend
	if len(ctx.Args) > 0 {
		plugBends := prov.NewPlugBackends(&ctx.Plugs)
		bend, err = plugBends.Provide(ctx.Args[0], pr.Project)
		if err != nil {
			return fmt.Errorf("open data: %v", err)
		}
		ctx.Args = ctx.Args[1:]
	}
	ctx.Wrap = func(ctx *xps.CmdCtx, env exp.Env) exp.Env {
		q := qry.New(pr.Reg, env, bend)
		return &qry.Doc{Qry: q}
	}
	return &xps.CmdRedir{Cmd: "repl"}
}

func status(ctx *xps.CmdCtx) error {
	pr, err := daql.LoadProject(ctx.Dir)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(os.Stdout)
	pr.Status(&bfr.P{Writer: b})
	return b.Flush()
}

func commit(ctx *xps.CmdCtx) error {
	pr, err := daql.LoadProject(ctx.Dir)
	if err != nil {
		return err
	}
	err = pr.Commit(strings.Join(ctx.Args, " "))
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

func graph(ctx *xps.CmdCtx) error {
	pr, ss, err := daql.LoadProjectSchemas(ctx.Dir, ctx.Args)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(os.Stdout)
	err = GraphSchemas(&bfr.P{Writer: b}, pr, ss)
	if err != nil {
		return err
	}
	return b.Flush()
}

func genGo(ctx *xps.CmdCtx) error {
	pr, ss, err := daql.LoadProjectSchemas(ctx.Dir, ctx.Args)
	if err != nil {
		return err
	}
	ppkg, err := xps.GoModPath(pr.Dir)
	if err != nil {
		return err
	}
	pkgs := gengo.DefaultPkgs()
	for _, s := range ss {
		if gen.Nogen(s) {
			continue
		}
		out := filepath.Join(daql.SchemaPath(pr, s), fmt.Sprintf("%s_gen.go", s.Name))
		b := gengo.NewGenPkgs(pr.Project, s.Name, path.Join(ppkg, s.Name), pkgs)
		err := gengo.WriteSchemaFile(b, out, s)
		if err != nil {
			return err
		}
		fmt.Println(out)
	}
	return nil
}
