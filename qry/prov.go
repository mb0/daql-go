package qry

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"xelf.org/daql/dom"
	"xelf.org/daql/mig"
	"xelf.org/xelf/lit"
)

var bends sync.Map

// Provider produces backends based on an uri.
type Provider interface {
	Provide(uri string, pr *dom.Project) (Backend, error)
}

func RegisterProvider(prov Provider, schemes ...string) Provider {
	for _, s := range schemes {
		bends.Store(s, prov)
	}
	return prov
}

func LoadProvider(scheme string) Provider {
	a, _ := bends.Load(scheme)
	prov, _ := a.(Provider)
	return prov
}

var Prov = RegisterProvider(dsetProvider{}, "file")

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
