package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/daql/gen/gengo"
	"xelf.org/xelf/lit"
)

const usage = `usage: daql [-dir=<path>] <command> [<args>]

Configuration flags:

   -dir        The project directory where the project file can be found.
               If this flag is not set, the current directory and its parents will be searched.

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

func genGen(gen string, args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	ss, err := filterSchemas(pr, args)
	if err != nil {
		return err
	}
	switch gen {
	case "gengo":
		return gogen(pr, ss)
	}
	return fmt.Errorf("no generator found for %s", gen)
}

func gogen(pr *Project, ss []*dom.Schema) error {
	ppkg, err := gopkg(pr.Dir)
	if err != nil {
		return err
	}
	pkgs := gengo.DefaultPkgs()
	for _, s := range pr.Schemas {
		if nogen(s) {
			continue
		}
		out := filepath.Join(schemaPath(pr, s), fmt.Sprintf("%s_gen.go", s.Name))
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

type Project struct {
	Dir  string // project directory
	Path string // rel path to dir
	*dom.Project
}

func project() (*Project, error) {
	path, err := dom.DiscoverProject(*dirFlag)
	if err != nil {
		return nil, fmt.Errorf("discover project: %v", err)
	}
	p, err := dom.OpenProject(path)
	if err != nil {
		return nil, err
	}
	return &Project{filepath.Dir(path), path, p}, nil
}

func filterSchemas(pr *Project, names []string) ([]*dom.Schema, error) {
	if len(names) == 0 {
		return pr.Schemas, fmt.Errorf("requires list of schema names")
	}
	ss := make([]*dom.Schema, 0, len(names))
	for _, name := range names {
		s := pr.Schema(name)
		if s == nil {
			return nil, fmt.Errorf("schema %q not found", name)
		}
		ss = append(ss, s)
	}
	return ss, nil
}
func gotool(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func gopkg(dir string) (string, error) {
	b, err := gotool(dir, "list", "-m")
	if err != nil {
		return "", fmt.Errorf("gopkg for %s: %v", dir, err)
	}
	return strings.TrimSpace(string(b)), nil
}

func schemaPath(pr *Project, s *dom.Schema) string {
	v, _ := s.Extra["file"]
	path, _ := lit.ToStr(v)
	if path != "" {
		return filepath.Join(pr.Dir, filepath.Dir(string(path)))
	}
	return pr.Dir
}
