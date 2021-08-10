package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	"xelf.org/daql/dom/domtest"
	"xelf.org/daql/qry"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib/extlib"
	"xelf.org/xelf/lit"
)

func repl(args []string) error {
	reg := &lit.Reg{}
	// use fixture and memory backend for now
	fix, err := domtest.ProdFixture()
	if err != nil {
		return fmt.Errorf("parse fixture: %v", err)
	}
	membed := &qry.MemBackend{Reg: reg, Project: &fix.Project}
	prodsch := fix.Schema("prod")
	for _, kv := range fix.Fix.Keyed {
		err = membed.Add(prodsch.Model(kv.Key), kv.Val.(*lit.List))
		if err != nil {
			return fmt.Errorf("prepare backend, add %s: %v", kv.Key, err)
		}
	}
	// TODO use the backup and a temporary database if we have a dataset argument
	// otherwise try the configured db
	lin := liner.NewLiner()
	defer lin.Close()
	readReplHistory(lin)
	lin.SetMultiLineMode(true)
	var buf bytes.Buffer
	var multi bool
	q := qry.New(reg, extlib.Std, membed)
	for {
		prompt := "> "
		if multi = buf.Len() > 0; multi {
			prompt = "… "
		}
		got, err := lin.Prompt(prompt)
		if err != nil {
			buf.Reset()
			if err == io.EOF {
				writeReplHistory(lin)
				fmt.Println()
				return nil
			}
			log.Printf("unexpected error reading prompt: %v", err)
			continue
		}
		got = strings.TrimSpace(got)
		if got == "" {
			continue
		}
		if multi {
			buf.WriteByte(' ')
		}
		buf.WriteString(got)
		lin.AppendHistory(buf.String())
		el, err := exp.Read(reg, &buf, "")
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			buf.Reset()
			log.Printf("error parsing %s: %v", got, err)
			continue
		}
		buf.Reset()
		l, err := q.ExecExp(el, nil)
		if err != nil {
			log.Printf("error resolving %s: %v", got, err)
			continue
		}
		fmt.Printf("= %s\n\n", l)
	}
	return nil
}

func replHistoryPath() string {
	path, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "daql/repl.history")
}

func readReplHistory(lin *liner.State) {
	path := replHistoryPath()
	if path == "" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	_, err = lin.ReadHistory(f)
	if err != nil {
		log.Printf("error reading repl history file %q: %v\n", path, err)
	}
}

func writeReplHistory(lin *liner.State) {
	path := replHistoryPath()
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("error creating dir for repl history %q: %v\n", dir, err)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("error creating file for repl history %q: %v\n", path, err)
		return
	}
	defer f.Close()
	_, err = lin.WriteHistory(f)
	if err != nil {
		log.Printf("error writing repl history file %q: %v\n", path, err)
	}
}
