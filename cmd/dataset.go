package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"xelf.org/dapgx"
	"xelf.org/dapgx/qrypgx"
	"xelf.org/daql/mig"
	"xelf.org/daql/qry"
	"xelf.org/xelf/lit"
)

type Data struct {
	URL *url.URL
	qry.Backend
	mig.Dataset
}

func OpenData(pr *Project, uri string) (*Data, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	log.Printf("got uri scheme %s %s", u.Scheme, uri)
	switch u.Scheme {
	case "postgres", "postgresql":
		// postgres://localhost:123/daql
		// postgres:///daql
		// postgres:///daql?host=/opt/run/postgresql
		db, err := dapgx.Open(context.Background(), uri, nil)
		if err != nil {
			return nil, err
		}
		bend := qrypgx.New(db, pr.Project)
		return &Data{u, bend, bend}, nil
	case "", "file":
		dset, err := mig.ReadDataset(u.Path)
		if err != nil {
			return nil, err
		}
		bend := &qry.MemBackend{Reg: pr.Reg, Project: pr.Project}
		for _, key := range dset.Keys() {
			m := pr.Project.Model(key)
			stream, err := dset.Stream(key)
			if err != nil {
				log.Printf("stream error: %v", err)
				continue
			}
			var vals []lit.Val
			v, err := stream.Scan()
			for err != nil {
				vals = append(vals, v)
				v, err = stream.Scan()
			}
			log.Printf("stream error: %v", err)
			err = bend.Add(m, &lit.List{Reg: pr.Reg, Vals: vals})
			if err != nil {
				return nil, fmt.Errorf("prepare backend, add %s: %v", key, err)
			}
		}
		return &Data{u, bend, dset}, nil
	}
	return nil, fmt.Errorf("no resolver for data uri scheme %s", uri)
}
