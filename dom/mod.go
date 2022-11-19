package dom

import (
	"fmt"
	"strings"

	"xelf.org/xelf/lit"
)

var Dom *Schema

func init() {
	var err error
	Dom, err = ReadSchema(domReg, strings.NewReader(RawSchema()), "go:xelf.org/daql/dom/dom.daql")
	if err != nil {
		panic(fmt.Errorf("could not read dom schema: %s", err))
	}
}

func SetupReg(reg *lit.Reg) {
	reg.AddFrom(domReg)
}
