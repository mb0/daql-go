package gengo

import (
	"strings"

	"xelf.org/daql/dom"
	"xelf.org/daql/gen"
)

func DefaultPkgs() map[string]string {
	return map[string]string{
		"cor": "xelf.org/xelf/cor",
		"lit": "xelf.org/xelf/lit",
		"typ": "xelf.org/xelf/typ",
		"exp": "xelf.org/xelf/exp",
		"dom": "xelf.org/daql/dom",
		"mig": "xelf.org/daql/mig",
	}
}

func NewGen(pr *dom.Project, pkg, path string) *gen.Gen {
	return NewGenPkgs(pr, pkg, path, DefaultPkgs())
}
func NewGenPkgs(pr *dom.Project, pkg, path string, pkgs map[string]string) *gen.Gen {
	pkgs[pkg] = path
	return &gen.Gen{
		Project: pr, Pkg: path,
		Pkgs:   pkgs,
		Header: "// generated code\n\n",
	}
}

// Import takes a qualified name of the form 'pkg.Decl', looks up a path from context packages
// map if available, otherwise the name is used as path. If the package path is the same as the
// context package it returns the 'Decl' part. Otherwise it adds the package path to the import
// list and returns a substring starting with last package path segment: 'pkg.Decl'.
func Import(c *gen.Gen, name string) string {
	ptr := name[0] == '*'
	if ptr {
		name = name[1:]
	}
	idx := strings.LastIndexByte(name, '.')
	var ns string
	if idx > -1 {
		ns = name[:idx]
	}
	if ns != "" && c != nil {
		if path, ok := c.Pkgs[ns]; ok {
			ns = path
		}
		if ns != c.Pkg {
			c.Imports.Add(ns)
		} else {
			name = name[idx+1:]
		}
	}
	if idx := strings.LastIndexByte(name, '/'); idx != -1 {
		name = name[idx+1:]
	}
	if ptr {
		name = "*" + name
	}
	return name
}
