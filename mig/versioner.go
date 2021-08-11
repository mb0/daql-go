package mig

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"xelf.org/daql/dom"
	"xelf.org/xelf/lit"
)

// NewVersioner returns a new versioner based on the given manifest.
func NewVersioner(mf Manifest) Versioner {
	mv := make(Versioner, len(mf))
	for i, v := range mf {
		key := v.Name
		if i == 0 {
			key = "_"
		}
		e := mv[key]
		if e == nil {
			mv[key] = &entry{old: v}
		} else if e.old.Vers < v.Vers {
			e.old = v
		}
	}
	return mv
}

// Versioner sets and returns node version details, usually based on the last recorded manifest.
type Versioner map[string]*entry

// Manifest returns a update manifest for Project.
func (mv Versioner) Manifest(pr *dom.Project) (Manifest, error) {
	_, err := mv.Version(pr)
	if err != nil {
		return nil, err
	}
	mf := make(Manifest, 0, len(mv))
	for _, e := range mv {
		if e.cur.Vers != "" {
			mf = append(mf, e.cur)
		} else {
			mf = append(mf, e.old)
		}
	}
	return mf.Sort(), nil
}

// Version sets and returns the node version details or an error.
func (mv Versioner) Version(n dom.Node) (res Version, err error) {
	res.Name = n.Qualified()
	key := res.Name
	if key[0] == '_' {
		key = "_"
	}
	e := mv[key]
	var resv Vers
	if e == nil {
		// use empty version?
	} else if e.cur.Vers != "" { // we already did the work
		return e.cur, nil
	} else if e.old.Vers != "" {
		resv, err = ParseVers(e.old.Vers)
		if err != nil {
			return res, err
		}
	} else {
		return res, fmt.Errorf("internal manifest error: inconsistent state")
	}
	hm := sha256.New()
	hp := sha256.New()
	switch d := n.(type) {
	case *dom.Model:
		fmt.Fprintf(hm, "%s:%s\n", d.Name, d.Kind)
		if d.Object != nil {
			fmt.Fprintf(hp, " order:%v\n", d.Object.OrderBy)
			for _, x := range d.Object.Indices {
				fmt.Fprintf(hm, " index:%s:%v:%v\n", x.Name, x.Keys, x.Unique)
			}
		}
		hashExtra(hm, hp, d.Extra)
		for _, e := range d.Elems {
			fmt.Fprintf(hm, "  %s:%s:%d:%d:%s\n", e.Name, e.Type, e.Val, e.Bits, e.Ref)
			hashExtra(hm, hp, e.Extra)
		}
	case *dom.Schema:
		fmt.Fprintf(hm, "%s\n", res.Name)
		hashExtra(hm, hp, d.Extra)
		for _, m := range d.Models {
			err := mv.hashNode(hm, hp, m)
			if err != nil {
				return res, err
			}
		}
	case *dom.Project:
		fmt.Fprintf(hm, "%s\n", res.Name)
		hashExtra(hm, hp, d.Extra)
		for _, s := range d.Schemas {
			err := mv.hashNode(hm, hp, s)
			if err != nil {
				return res, err
			}
		}
	default:
		return res, fmt.Errorf("unexpected node type %T", n)
	}
	res.Minor = hex.EncodeToString(hm.Sum(nil))
	res.Patch = hex.EncodeToString(hp.Sum(nil))
	if e == nil {
		res.Vers = resv.String()
		mv[key] = &entry{cur: res}
	} else if res.Minor != e.old.Minor || res.Patch != e.old.Patch {
		if res.Minor != e.old.Minor {
			resv.Minor++
			resv.Patch = 0
		} else {
			resv.Patch++
		}
		res.Vers = resv.String()
		e.cur = res
	} else {
		res = e.old
		e.cur = res
	}
	return res, nil
}

type entry struct {
	old Version
	cur Version
}

func (mv Versioner) hashNode(hm, hp io.Writer, n dom.Node) error {
	v, err := mv.Version(n)
	if err != nil {
		return err
	}
	hm.Write([]byte(v.Minor))
	hp.Write([]byte(v.Patch))
	return nil
}

func hashExtra(hm, hp io.Writer, d map[string]lit.Val) {
	if len(d) == 0 {
		return
	}
	for k, v := range d {
		switch k {
		case "doc":
			continue
		case "backup":
			fmt.Fprintf(hm, "    %s:%s\n", k, v)
		default:
			fmt.Fprintf(hp, "    %s:%s\n", k, v)
		}
	}
}
