package cmd

import (
	"fmt"
	"net/url"

	"xelf.org/daql/mig"
	"xelf.org/daql/qry"
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
	if u.Scheme == "" {
		u.Scheme = "file"
	}
	prov := qry.LoadProvider(u.Scheme)
	if prov == nil {
		return nil, fmt.Errorf("no backend provider for %s", u.Scheme)
	}
	bend, err := prov.Provide(uri, pr.Project)
	if err != nil {
		return nil, fmt.Errorf("failed creating backend for %s: %s", u.Scheme, err)
	}
	dset, _ := bend.(mig.Dataset)
	return &Data{u, bend, dset}, nil
}
