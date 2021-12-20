package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/peterh/liner"
	"xelf.org/daql/qry"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

func ReplHistoryPath() string {
	path, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "daql/repl.history")
}

type Repl struct {
	Reg  *lit.Reg
	Bend qry.Backend
	*liner.State
	Hist string
}

func NewRepl(reg *lit.Reg, bend qry.Backend, hist string) *Repl {
	lin := liner.NewLiner()
	lin.SetMultiLineMode(true)
	return &Repl{Reg: reg, Bend: bend, State: lin, Hist: hist}
}

func (r Repl) Run() {
	r.readHistory()
	defer r.Close()
	var raw []byte
	q := qry.New(r.Reg, extlib.Std, r.Bend)
	for {
		prompt := "> "
		if len(raw) > 0 {
			prompt = "… "
		}
		got, err := r.Prompt(prompt)
		if err != nil {
			if err == io.EOF {
				r.writeHistory()
				fmt.Println()
				return
			}
			raw = raw[:0]
			log.Printf("unexpected error reading prompt: %v", err)
			continue
		}
		raw = append(raw, got...)
		el, err := exp.Read(r.Reg, bytes.NewReader(raw), "")
		if err != nil {
			if errors.Is(err, io.EOF) {
				raw = append(raw, '\n')
				continue
			}
			log.Printf("error parsing %s: %v", got, err)
			r.AppendHistory(string(raw))
			raw = raw[:0]
			continue
		}
		r.AppendHistory(bfr.String(el))
		raw = raw[:0]
		l, err := q.ExecExp(el, nil)
		if err != nil {
			log.Printf("error resolving %s: %v", got, err)
			continue
		}
		fmt.Printf("= %s\n\n", bfr.String(l))
	}
}

func (r *Repl) readHistory() {
	if r.Hist == "" {
		return
	}
	f, err := os.Open(r.Hist)
	if err != nil {
		log.Printf("error reading repl history file %q: %v\n", r.Hist, err)
		return
	}
	defer f.Close()
	_, err = r.ReadHistory(f)
	if err != nil {
		log.Printf("error reading repl history file %q: %v\n", r.Hist, err)
	}
}

func (r *Repl) writeHistory() {
	if r.Hist == "" {
		return
	}
	dir := filepath.Dir(r.Hist)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("error creating dir for repl history %q: %v\n", dir, err)
		return
	}
	f, err := os.Create(r.Hist)
	if err != nil {
		log.Printf("error creating file for repl history %q: %v\n", r.Hist, err)
		return
	}
	defer f.Close()
	_, err = r.WriteHistory(f)
	if err != nil {
		log.Printf("error writing repl history file %q: %v\n", r.Hist, err)
	}
}
