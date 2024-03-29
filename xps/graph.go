package main

import (
	"xelf.org/daql"
	"xelf.org/daql/dom"
	"xelf.org/xelf/bfr"
)

func GraphSchemas(p *bfr.P, pr *daql.Project, ss []*dom.Schema) error {
	p.Fmt("// install graphviz to turn the output into an image. for example:\n")
	p.Fmt("// $ daql graph | dot -Tsvg > graph.svg && open graph.svg\n")
	p.Fmt("digraph %s {\ngraph [rankdir=LR]\n", pr.Name)
	for _, s := range ss {
		p.Fmt("subgraph cluster_%s {\n", s.Name)
		p.Fmt("node[shape=record]\ncolor=gray\nlabel=\"%s\"\n", s.Name)
		for _, m := range s.Models {
			p.Fmt("\"%s\" [label=\"%s\"]\n", m.Qualified(), m.Name)
		}
		p.Fmt("}\n")
	}
	rels, err := dom.Relate(pr.Project)
	if err != nil {
		return err
	}
	for _, s := range ss {
		for _, m := range s.Models {
			key := m.Qualified()
			rel := rels[key]
			if rel == nil {
				continue
			}
			for _, r := range rel.Out {
				if r.Via.Model == nil {
					p.Fmt("\"%s\"->\"%s\"\n", key, r.B.Qualified())
				}
			}
		}
	}
	return p.Fmt("}\n")
}
