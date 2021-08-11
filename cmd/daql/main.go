package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"xelf.org/daql/cmd"
	"xelf.org/daql/dom"
	"xelf.org/daql/gen/gengo"
)

const usage = `usage: daql [-dir=<path>] <command> [<args>]

Configuration flags:

   -dir        The project directory where the project file can be found.
               If this flag is not set, the current directory and its parents will be searched.

Model versioning commands
   status      Prints the model version manifest for the current project

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
	pr.Status(b)
	return b.Flush()
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
	for _, s := range pr.Schemas {
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
