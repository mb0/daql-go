package domtest

import "xelf.org/xelf/lit"

const ProdRaw = `(schema prod
(Cat; topic;
	ID:int
	Name:str
)
(Prod; topic;
	ID:int
	Name:str
	@Cat.ID
)
(Label; topic;
	ID:int
	Name:str
	Tmpl:raw
)
)`

type Cat struct {
	ID   int
	Name string
}

type Prod struct {
	ID   int
	Name string
	Cat  int
}

type Label struct {
	ID   int
	Name string
	Tmpl []byte
}

const ProdFixRaw = `{
	cat:[
		[25 'y']
		[2  'b']
		[3  'c']
		[1  'a']
		[4  'd']
		[26 'z']
		[24 'x']
	]
	prod:[
		[25 'Y' 1]
		[2  'B' 2]
		[3  'C' 3]
		[1  'A' 3]
		[4  'D' 2]
		[26 'Z' 1]
	]
	label:[
		[1 'M' 'foo']
		[2 'N' 'bar']
		[3 'O' 'spam']
		[4 'P' 'egg']
	]
}`

func ProdFixture(reg *lit.Reg) (*Fixture, error) { return New(reg, ProdRaw, ProdFixRaw) }
