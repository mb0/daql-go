package domtest

import (
	"testing"

	"xelf.org/xelf/lit"
)

func TestDomtest(t *testing.T) {
	reg := lit.NewRegs()
	_, err := ProdFixture(reg)
	if err != nil {
		t.Fatalf("prod fixture error: %v", err)
	}
	_, err = PersonFixture(reg)
	if err != nil {
		t.Fatalf("person fixture error: %v", err)
	}
}
