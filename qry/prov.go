package qry

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"

	"xelf.org/daql/dom"
	"xelf.org/daql/mig"
	"xelf.org/xelf/lit"
)

// Provider produces backends based on an uri.
type Provider interface {
	Provide(uri string, pr *dom.Project) (Backend, error)
}

var Backends = &Registry{}

type Registry struct {
	sync.Mutex
	bends map[string]Provider
}

func (r *Registry) Register(prov Provider, schemes ...string) Provider {
	r.Lock()
	defer r.Unlock()
	if r.bends == nil {
		r.bends = make(map[string]Provider)
	}
	for _, s := range schemes {
		r.bends[s] = prov
	}
	return prov
}

var ErrNoProvider = fmt.Errorf("no backend provider found")

func (r *Registry) Provide(uri string, pr *dom.Project) (Backend, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "file"
	}
	r.Lock()
	prov := r.bends[u.Scheme]
	defer r.Unlock()
	if prov == nil {
		return nil, ErrNoProvider
	}
	return prov.Provide(uri, pr)
}

var Prov = Backends.Register(dsetProvider{}, "file")

type dsetProvider struct{}

func (dsetProvider) Provide(uri string, pro *dom.Project) (Backend, error) {
	dset, err := mig.ReadDataset(uri)
	if err != nil {
		return nil, err
	}
	return NewDsetBackend(pro, dset)
}

func NewDsetBackend(pro *dom.Project, dset mig.Dataset) (*MemBackend, error) {
	b := NewMemBackend(pro, dset.Vers())
	for _, key := range dset.Keys() {
		m := pro.Model(key)
		if m == nil {
			continue
		}
		stream, err := dset.Stream(key)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return nil, fmt.Errorf("stream error: %v", err)
		}
		var vals lit.Vals
		v, err := stream.Scan()
		for err == nil {
			vals = append(vals, v)
			v, err = stream.Scan()
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("stream error: %v", err)
		}
		err = b.Add(m, &vals)
		if err != nil {
			return nil, fmt.Errorf("prepare backend, add %s: %v", key, err)
		}
	}
	return b, nil
}
