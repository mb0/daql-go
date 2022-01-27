package mig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/daql/log"
	"xelf.org/xelf/lit"
)

var NoVers = Vers{}

// ReadVersion returns a version read from r or and error.
func ReadVersion(r io.Reader) (v Version, err error) {
	err = json.NewDecoder(r).Decode(&v)
	return v, err
}

// WriteTo writes the version to w and returns the written bytes or an error.
func (v Version) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(v)
	if err != nil {
		return 0, err
	}
	return b.WriteTo(w)
}

// Vers represents a parsed version string with int fields for major, minor and patch versions.
type Vers struct {
	Major, Minor, Patch int
}

// ParseVers parses str and returns the result or an error.
func ParseVers(str string) (v Vers, err error) {
	if str[0] == 'v' {
		str = str[1:]
	}
	for i, part := range strings.SplitN(str, ".", 4) {
		num, err := strconv.Atoi(part)
		if err != nil {
			return v, err
		}
		switch i {
		case 0:
			v.Major = num
		case 1:
			v.Minor = num
		case 2:
			v.Patch = num
		default:
			log.Debug("unexpected version rest %s", str)
		}
	}
	return v, nil
}

func (v Vers) String() string { return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch) }

// Record consists of a project definition and its manifest at one point in time.
// A record's path can be used to look up migration rules and scripts.
type Record struct {
	Path string `json:"path"` // record path relative to history folder
	*dom.Project
	Manifest `json:"manifest"`
}

// ReadProject reads the current project's unrecorded definition and manifest or an error.
//
// The returned record represent the current malleable project state, and may contain unrecorded
// changes and preliminary versions, not representing the eventually recorded version definition.
func ReadRecord(reg *lit.Reg, path string) (res Record, err error) {
	res.Project, err = dom.OpenProject(reg, path)
	if err != nil {
		return res, err
	}
	hdir := historyPath(res.Project, path)
	err = readFile(filepath.Join(hdir, "manifest.json"),
		func(r io.Reader) (err error) {
			res.Manifest, err = ReadManifest(r)
			return err
		})
	if err != nil {
		return res, err
	}
	res.Manifest, err = res.Update(res.Project)
	return res, err
}

func (r *Record) Version() Version { return *r.First() }
