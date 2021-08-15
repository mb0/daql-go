package mig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xelf.org/daql/dom"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/ext"
	"xelf.org/xelf/lit"
)

var ErrNoHistory = fmt.Errorf("no history")
var ErrNoChanges = fmt.Errorf("no changes")

// History provides project records.
type History interface {
	Path() string
	Curr() Record
	Last() Manifest
	Versions() []Version
	Manifest(vers string) (Manifest, error)
	Record(vers string) (Record, error)
	Commit(slug string) error
}

// ReadHistory returns the prepared project history based on a project path or an error.
//
// The history folder path defaults to '$project/hist', but can changed in the project definition.
// The folder contains a link to, or copy of the last recorded manifest and record folders each
// containing a project definition and manifest file, encoded as optionally gzipped JSON streams.
// The record folders can also contain migration rules and scripts, to migration data to that
// version. The history folder acts as staging area for migrations for unrecorded project changes.
// Record folders can have any name starting with a 'v', the daql tool uses the padded version
// number and an optional record note that only acts as memory aid. The actual record version should
// always be read from the included manifest file.
func ReadHistory(reg *lit.Reg, path string) (_ History, err error) {
	h := &hist{reg: reg}
	h.path, err = dom.DiscoverProject(path)
	if err != nil {
		return nil, fmt.Errorf("no project file found for %q: %v", path, err)
	}
	h.curr.Project, err = dom.OpenProject(reg, h.path)
	if err != nil {
		return nil, fmt.Errorf("resolve project %q: %v", h.path, err)
	}
	h.hdir = historyPath(h.curr.Project, h.path)
	dir, err := os.Open(h.hdir)
	if err != nil {
		h.curr.Manifest, err = h.curr.Manifest.Update(h.curr.Project)
		if err != nil {
			return nil, err
		}
		return h, ErrNoHistory
	}
	defer dir.Close()
	h.recs = make([]rec, 0, 8)
	fis, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() {
			if !strings.HasPrefix(name, "manifest.") || !isJsonStream(name) {
				continue
			}
			err = readFile(filepath.Join(h.hdir, name), func(r io.Reader) (err error) {
				h.curr.Manifest, err = ReadManifest(r)
				return err
			})
			if err != nil {
				return nil, fmt.Errorf("reading %s: %v", name, err)
			}
		} else {
			if !strings.HasPrefix(name, "v") {
				continue
			}
			mpath := filepath.Join(h.hdir, name, "manifest.json")
			err = readFile(mpath, func(r io.Reader) error {
				mf, err := ReadManifest(r)
				if err == nil {
					h.recs = append(h.recs, rec{name, mf})
				}
				return err
			})
			if err != nil {
				return nil, fmt.Errorf("reading %s: %v", mpath, err)
			}
		}
	}
	sort.Slice(h.recs, func(i, j int) bool {
		return h.recs[i].First().Vers < h.recs[j].First().Vers
	})
	if len(h.recs) > 0 {
		lst := h.recs[len(h.recs)-1]
		v, _ := ParseVers(h.curr.First().Vers)
		if v == NoVers {
			h.curr.Manifest = lst.Manifest
		} else if lv, _ := ParseVers(lst.First().Vers); v != lv {
			return nil, fmt.Errorf("inconsistent history manifest version %d != %d",
				v, lv)
		}
	}
	h.curr.Manifest, err = h.curr.Manifest.Update(h.curr.Project)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func historyPath(pr *dom.Project, path string) string {
	rel := "hist"
	if v, err := pr.Extra.Key("hist"); err == nil {
		if c, err := lit.ToStr(v); err == nil {
			rel = string(c)
		}
	}
	return filepath.Join(filepath.Dir(path), rel)
}

func isJsonStream(path string) bool {
	return strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".json.gz")
}

type hist struct {
	reg  *lit.Reg
	path string
	hdir string
	curr Record
	recs []rec
}

type rec struct {
	Path string
	Manifest
}

func (h *hist) Path() string { return h.path }
func (h *hist) Curr() Record { return h.curr }
func (h *hist) Last() Manifest {
	if n := len(h.recs); n > 0 {
		return h.recs[n-1].Manifest
	}
	return nil
}

func (h *hist) Versions() []Version {
	res := make([]Version, 0, len(h.recs))
	for _, r := range h.recs {
		res = append(res, *r.First())
	}
	return res
}

func (h *hist) Manifest(vers string) (Manifest, error) {
	r, ok := h.rec(vers)
	if !ok {
		return nil, fmt.Errorf("version not found")
	}
	return r.Manifest, nil
}
func (h *hist) rec(vers string) (rec, bool) {
	for _, r := range h.recs {
		if r.First().Vers == vers {
			return r, true
		}
	}
	return rec{}, false
}

func (h *hist) Record(vers string) (null Record, _ error) {
	r, ok := h.rec(vers)
	if !ok {
		return null, fmt.Errorf("version not found")
	}
	ppath := filepath.Join(h.hdir, r.Path, "project.json")
	pr, err := dom.OpenProject(h.reg, ppath)
	if err != nil {
		return null, err
	}
	return Record{r.Path, pr, r.Manifest}, nil
}

func (h *hist) Commit(slug string) error {
	c := h.curr.First()
	last := h.Last()
	l := last.First()
	if l != nil && c.Vers == l.Vers {
		return ErrNoChanges
	}
	err := os.MkdirAll(h.hdir, 0755)
	if err != nil {
		return fmt.Errorf("create history folder %s: %v", h.hdir, err)
	}
	now := time.Now()
	rec := h.curr
	// set recording date to all changed versions
	changes := rec.Diff(last)
	for i := range rec.Manifest {
		v := &rec.Manifest[i]
		if _, ok := changes[v.Name]; ok {
			v.Date = now
		}
	}
	err = writeFile(filepath.Join(h.hdir, "manifest.json"), func(w io.Writer) error {
		_, err := rec.Manifest.WriteTo(w)
		return err
	})
	if err != nil {
		return fmt.Errorf("write manifest.json: %v", err)
	}
	rec.Path = fmt.Sprintf("%s-%s", rec.First().Vers, now.Format("20060102"))
	if slug = cor.Keyify(slug); slug != "" {
		rec.Path = fmt.Sprintf("%s-%s", rec.Path, slug)
	}
	rdir := filepath.Join(h.hdir, rec.Path)
	err = os.MkdirAll(rdir, 0755)
	if err != nil {
		return fmt.Errorf("mkdir %s: %v", rec.Path, err)
	}
	err = writeFileGz(filepath.Join(rdir, "manifest.json.gz"), func(w io.Writer) error {
		_, err := rec.Manifest.WriteTo(w)
		return err
	})
	if err != nil {
		return fmt.Errorf("write manifest.json.gz: %v", err)
	}
	err = writeFileGz(filepath.Join(rdir, "project.json.gz"), func(w io.Writer) error {
		n, err := ext.NewNode(&lit.Reg{}, rec.Project)
		if err != nil {
			return err
		}
		b := bufio.NewWriter(w)
		err = n.Print(&bfr.P{Writer: b, JSON: true, Tab: "\t"})
		if err != nil {
			return err
		}
		return b.Flush()
	})
	if err != nil {
		return fmt.Errorf("write project.json.gz: %v", err)
	}
	// TODO also move migration rule and script files, as soon as we know how to spot them.
	return nil
}
