package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"xelf.org/daql/cmd"
	"xelf.org/daql/dom"
	"xelf.org/daql/gen/gengo"
	"xelf.org/daql/mig"
	"xelf.org/xelf/bfr"
)

const usage = `usage: daql [-dir=<path>] <command> [<args>]

Configuration flags:

   -dir        The project directory where the project file can be found.
               If this flag is not set, the current directory and its parents will be searched.

Model versioning commands
   status      Prints the model version manifest for the current project.
   commit      Writes the current project changes to the project history.
   graph       Prints a dot graph for the specified schema names. Use graphviz to render:
               $ daql graph | dot -Tsvg > graph.svg && open graph.svg

Code generation commands
   gengo       Generates go code for specific schemas specified in args.

Other commands
   help        Displays this help message.
   repl        Runs a read-eval-print-loop to explore xelf and daql.
               Repl currently only uses a test data model for queries.

`

var (
	dirFlag = flag.String("dir", ".", "project directory path")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	args := flag.Args()
	if len(args) == 0 {
		log.Printf("missing command\n\n")
		fmt.Print(usage)
		return
	}
	args = args[1:]
	var err error
	switch cmd := flag.Arg(0); cmd {
	case "status":
		err = status(args)
	case "commit":
		err = commit(args)
	case "graph":
		err = graph(args)
	case "gengo":
		err = genGen(cmd, args)
	case "repl":
		err = repl(args)
	case "help":
		if len(args) > 0 {
			// TODO print command help
		}
		fmt.Print(usage)
	default:
		log.Printf("unknown command: %s\n\n", cmd)
		fmt.Print(usage)
	}
	if err != nil {
		log.Fatalf("%s error: %+v\n", flag.Arg(0), err)
	}
}

func status(args []string) error {
	pr, err := cmd.LoadProject(*dirFlag)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(os.Stdout)
	pr.Status(&bfr.P{Writer: b})
	return b.Flush()
}

func commit(args []string) error {
	pr, err := cmd.LoadProject(*dirFlag)
	if err != nil {
		return err
	}
	err = pr.Commit(strings.Join(args, "_"))
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

func graph(args []string) error {
	pr, err := cmd.LoadProject(*dirFlag)
	if err != nil {
		return err
	}
	ss := pr.Schemas
	if len(args) > 0 {
		ss, err = pr.FilterSchemas(args...)
		if err != nil {
			return err
		}
	}
	b := bufio.NewWriter(os.Stdout)
	defer b.Flush()
	return cmd.GraphSchemas(&bfr.P{Writer: b}, pr, ss)
}

func genGen(gen string, args []string) error {
	pr, err := cmd.LoadProject(*dirFlag)
	if err != nil {
		return err
	}
	ss, err := pr.FilterSchemas(args...)
	if err != nil {
		return err
	}
	switch gen {
	case "gengo":
		return gogen(pr, ss)
	}
	return fmt.Errorf("no generator found for %s", gen)
}

func gogen(pr *cmd.Project, ss []*dom.Schema) error {
	ppkg, err := cmd.GoModPath(pr.Dir)
	if err != nil {
		return err
	}
	pkgs := gengo.DefaultPkgs()
	for _, s := range ss {
		if nogen(s) {
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

func nogen(s *dom.Schema) bool {
	l, ok := s.Extra["nogen"]
	return ok && !l.Nil()
}
