package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	xcmd "xelf.org/cmd"
	"xelf.org/dapgx"
	"xelf.org/dapgx/dompgx"
	"xelf.org/daql/cmd"
	"xelf.org/daql/dom"
	"xelf.org/daql/gen/gengo"
	"xelf.org/daql/mig"
	"xelf.org/dawui"
	"xelf.org/xelf/bfr"
)

const usage = `usage: daql [-dir=<path>] <command> [<args>]

Configuration flags:

   -dir        The project directory where the project file can be found.
               If not set, the current directory and its parents will be searched.

   -data       The dataset to use. Either a path to a backup or a connection string to a database.
               If not set the DAQL_DATA environment variable is tried.

Model versioning commands
   status      Prints the model version manifest for the current project.
   commit      Writes the current project changes to the project history.
   graph       Prints a dot graph for the specified schema names. Use graphviz to render:
               $ daql graph | dot -Tsvg > graph.svg && open graph.svg

Code generation commands
   gengo       Generates go code for specific schemas specified in args alongside the schema files.
   genpg       Generates postgresql schema definition for schemas specified in args to stdout.

Other commands
   help        Displays this help message.
   repl        Runs a read-eval-print-loop to explore xelf and daql.
               Repl uses the project and specified dataset or falls back on a test fixture.

`

var (
	dirFlag    = flag.String("dir", ".", "project directory path")
	dataFlag   = flag.String("data", "", "dataset either db uri or path to backup zip or folder")
	addrFlag   = flag.String("addr", "localhost:8090", "http address for webui")
	staticFlag = flag.String("static", "", "alternative static resources path for webui")
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
	case "gengo", "genpg":
		err = genGen(cmd, args)
	case "repl":
		err = repl(args)
	case "webui":
		var srv *dawui.Server
		srv, err = dawui.NewServer(*dirFlag, *dataFlag, *staticFlag)
		if err != nil {
			break
		}
		log.Printf("open server at http://%s", *addrFlag)
		err = http.ListenAndServe(*addrFlag, srv)
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
	case "genpg":
		return pggen(pr, ss)
	}
	return fmt.Errorf("no generator found for %s", gen)
}

func gogen(pr *cmd.Project, ss []*dom.Schema) error {
	ppkg, err := xcmd.GoModPath(pr.Dir)
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

func pggen(pr *cmd.Project, ss []*dom.Schema) error {
	b := bufio.NewWriter(os.Stdout)
	defer b.Flush()
	w := dapgx.NewWriter(b, pr.Project, nil, nil)
	w.WriteString(w.Header)
	w.WriteString("BEGIN;\n\n")
	for _, s := range ss {
		if nogen(s) {
			continue
		}
		err := dompgx.WriteSchema(w, s)
		if err != nil {
			return err
		}
	}
	w.WriteString("COMMIT;\n")
	return b.Flush()
}

func nogen(s *dom.Schema) bool {
	l, err := s.Extra.Key("nogen")
	return err == nil && !l.Nil()
}
