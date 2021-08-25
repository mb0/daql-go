package pol

import (
	"bufio"
	"fmt"
	"io"
)

// ReadRules returns a list of rules read from r or an error.
func ReadRules(r io.Reader) (res []Rule, err error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		b := sc.Bytes()
		if len(b) == 0 || b[0] == '#' {
			continue
		}
		var r Rule
		err := r.UnmarshalText(b)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	err = sc.Err()
	return res, err
}

// Rule is a simple source structure to build up a policy.
type Rule struct {
	Perm
	Top  string
	Role string
}

func (r Rule) String() string { b, _ := r.MarshalText(); return string(b) }
func (r Rule) MarshalText() ([]byte, error) {
	b := make([]byte, 0, 7+len(r.Top)+len(r.Role))
	pb, _ := r.Perm.MarshalText()
	b = append(b, pb...)
	b = append(b, '\t')
	b = append(b, r.Top...)
	b = append(b, '\t')
	b = append(b, r.Role...)
	return b, nil
}
func (r *Rule) UnmarshalText(b []byte) error {
	if len(b) < 5 || (b[0] != '-' && b[0] != '+' && b[0] != '@') {
		return fmt.Errorf("invalid rule %q", b)
	}
	// split tokens by spaces and or tabs
	ps := make([][]byte, 0, 3)
	var start int
	var ws bool
	for i, c := range b {
		if c == ' ' || c == '\t' {
			if !ws {
				ps = append(ps, b[start:i])
				ws = true
			}
		} else if ws {
			start = i
			ws = false
		}
	}
	if !ws {
		ps = append(ps, b[start:])
	}
	if len(ps) != 3 {
		return fmt.Errorf("invalid rule %q", b)
	}
	err := r.Perm.UnmarshalText(ps[0])
	if err != nil {
		return err
	}
	r.Top = string(ps[1])
	r.Role = string(ps[2])
	return nil
}

// Perm represents either a role association or a permitted or denied operation.
// Role associations have the value 0, denied op is -op and permitted op is itself.
type Perm int

func (p Perm) Op() Op {
	if p < 0 {
		p = -p
	}
	return Op(p)
}

func (p Perm) String() string { b, _ := p.MarshalText(); return string(b) }
func (p Perm) MarshalText() ([]byte, error) {
	if p == 0 {
		return []byte{'@'}, nil
	}
	b := make([]byte, 0, 5)
	if p < 0 {
		p = -p
		b = append(b, '-')
	} else {
		b = append(b, '+')
	}
	o := Op(p)
	if o == All {
		return append(b, 'a'), nil
	}
	if o&D != 0 {
		b = append(b, 'd')
	}
	if o&R != 0 {
		b = append(b, 'r')
	}
	if o&W != 0 {
		b = append(b, 'w')
	}
	if o&X != 0 {
		b = append(b, 'x')
	}
	return b, nil
}

func (p *Perm) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("no perm")
	}
Outer:
	switch k := b[0]; k {
	case '@':
		if len(b) == 1 {
			*p = 0
			return nil
		}
	case '-', '+':
		var m, o Op
		for _, r := range b[1:] {
			switch r {
			case 'a':
				o = All
			case 'd':
				o = D
			case 'r':
				o = R
			case 'w':
				o = W
			case 'x':
				o = X
			default:
				break Outer
			}
			if m&o != 0 {
				return fmt.Errorf("duplicate flag %q", r)
			}
			m |= o
		}
		if k == '-' {
			*p = -Perm(m)
		} else {
			*p = Perm(m)
		}
		return nil
	}
	return fmt.Errorf("unexpected perm %q", b)
}
