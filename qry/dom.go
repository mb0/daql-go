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
	var vals lit.Vals
	switch j.Ref {
	case "dom.model":
		vals = make([]lit.Val, 0, len(b.Schemas)*8)
		for _, s := range b.Schemas {
			for _, m := range s.Models {
				x, _ := p.Reg.Proxy(m)
				vals = append(vals, x)
			}
		}
	case "dom.schema":
		vals = make([]lit.Val, 0, len(b.Schemas))
		for _, s := range b.Schemas {
			x, _ := p.Reg.Proxy(s)
			vals = append(vals, x)
		}
	case "dom.project":
		vals = lit.Vals{lit.MustProxy(p.Reg, b.Project)}
	default:
		return nil, fmt.Errorf("dom backend: unexpected ref %s", j.Ref)
	}
	return execListQry(p, j, vals)
}
