package prov

import (
	"xelf.org/daql/dom"
	"xelf.org/daql/qry"
	"xelf.org/xelf/mod"
	"xelf.org/xelf/xps"
)

// PlugBackends wrapps a qry backend provider registry with a plugin list.
// It lazy-loads plugins that provide module source missing from the registry.
type PlugBackends struct {
	*qry.Registry
	*xps.Plugs
}

func NewPlugBackends(plugs *xps.Plugs) *PlugBackends {
	return &PlugBackends{qry.Backends, plugs}
}

func (r *PlugBackends) Provide(uri string, pr *dom.Project) (qry.Backend, error) {
	bend, err := r.Registry.Provide(uri, pr)
	if err == qry.ErrNoProvider {
		p := provPlug(r.All, mod.ParseLoc(uri).Proto())
		if p != nil && p.Plugin == nil {
			if err = p.Load(); err != nil {
				return nil, err
			}
			bend, err = r.Registry.Provide(uri, pr)
		}
	}
	return bend, err
}

func provPlug(all map[string]*xps.Plug, proto string) *xps.Plug {
	for _, p := range all {
		for _, bend := range p.CapList("bend") {
			if bend == proto {
				return p
			}
		}
	}
	return nil
}
