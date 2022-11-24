package domtest

import (
	"xelf.org/daql/dom"
	"xelf.org/daql/qry"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/mod"
)

var Prov = qry.RegisterProvider(provider{}, "domtest")

type provider struct{}

func (provider) Provide(uri string, pro *dom.Project) (_ qry.Backend, err error) {
	name := mod.ParseLoc(uri).Path()
	var fix *Fixture
	reg := &lit.Regs{}
	switch name {
	case "person":
		fix, err = PersonFixture(reg)
	case "prod":
		fix, err = ProdFixture(reg)
	}
	if err != nil {
		return nil, err
	}
	pro.Schemas = append(pro.Schemas, fix.Schemas...)
	return qry.NewDsetBackend(pro, fix)
}
