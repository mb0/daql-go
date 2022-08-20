package qry

import (
	"fmt"

	"xelf.org/daql/dom"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
)

type DomBackend struct {
	*dom.Project
}

func (b *DomBackend) Proj() *dom.Project { return b.Project }
func (b *DomBackend) Exec(p *exp.Prog, j *Job) (*exp.Lit, error) {
	var list *lit.List
	switch j.Ref {
	case "dom.model":
		list = &lit.List{Vals: make([]lit.Val, 0, len(b.Schemas)*8)}
		for _, s := range b.Schemas {
			for _, m := range s.Models {
				x, _ := p.Reg.Proxy(m)
				if len(list.Vals) == 0 {
					list.El = x.Type()
				}
				list.Vals = append(list.Vals, x)
			}
		}
	case "dom.schema":
		list = &lit.List{Vals: make([]lit.Val, 0, len(b.Schemas))}
		for _, s := range b.Schemas {
			x, _ := p.Reg.Proxy(s)
			if len(list.Vals) == 0 {
				list.El = x.Type()
			}
			list.Vals = append(list.Vals, x)
		}
	case "dom.project":
		x, _ := p.Reg.Proxy(b.Project)
		list = &lit.List{El: x.Type(), Vals: []lit.Val{x}}
	}
	if list == nil {
		return nil, fmt.Errorf("dom backend: not implemented")
	}
	return execListQry(p, j, list)
}
