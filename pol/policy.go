// Package pol offers a role based permission system with a rules based implementation.
package pol

// Policy is an interface for a simple permissions checker.
type Policy interface {
	// Police permits role to execute the given actions or returns an error.
	// Role names must not contain white space or '*' characters.
	Police(role string, acts ...Action) error
}

// Action is described by an operation and a topic.
type Action struct {
	Op
	// Top is the topic and most commonly a model name. Special events should use custom topics.
	// Topic names must not contain white space or '*' characters.
	Top string
}

// Op describes an operation as a bitset with bits for read, write, execute and delete.
// These operations map to qry reads and event commands new, mod, del for dom model topics.
// Custom topics are free to have another interpretation but usually only use the X op.
type Op uint

const (
	// X op is used for creating new model objects or executing custom actions.
	X Op = 1 << iota
	// W op is for writing/modifying model objects.
	W
	// R op is for reading/querying model objects.
	R
	// D op is for deleting model objects.
	D
	All = D | R | W | X
)

func (o Op) String() string {
	var res string
	if o&D != 0 {
		res += "d"
	}
	if o&R != 0 {
		res += "r"
	}
	if o&W != 0 {
		res += "w"
	}
	if o&X != 0 {
		res += "x"
	}
	return res
}
