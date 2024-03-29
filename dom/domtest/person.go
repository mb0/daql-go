package domtest

import (
	"time"

	"xelf.org/xelf/lit"
)

const PersonRaw = `(schema person
(Group; topic;
	ID:int
	Name:str
)
(Gender:enum
	D;
	F;
	M;
)
(Person; topic;
	ID:int
	Name:str
	@Gender
	(Family:@Group.ID)
)
(Member; topic;
	ID:int
	@Person.ID
	@Group.ID
	Joined:time
))`

type Group struct {
	ID   int
	Name string
}

type Person struct {
	ID     int
	Name   string
	Gender string
	Family int
}

type Member struct {
	ID     int
	Person int
	Group  int
	Joined time.Time
}

const PersonFixRaw = `{
	group:[
		[1  'Schnabels']
		[2  'Starkeys']
		[3  'Beatles']
		[4  'Gophers']
	]
	person:[
		[1  'Martin' 1 'm']
		[2  'Ringo'  2 'm']
		[3  'Rob'    4 'm']
		[4  'Corp'   0 'd']
	]
	member:[
		[1 1 1 '1983-11-07']
		[2 2 2 '1940-07-07']
		[3 2 3 '1962-08-01']
		[4 1 4 '2012-02-20']
		[5 3 4 '2009-10-10']
	]
}`

func PersonFixture(reg *lit.Regs) (*Fixture, error) { return New(reg, PersonRaw, PersonFixRaw) }
