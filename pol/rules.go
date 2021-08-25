package pol

import (
	"fmt"
	"io"
)

// RulePolicy is a Policy based on a set of rules and supports group roles and wildcards.
// The implementation is not thread safe for writes. Either lock access or build anew and swap out.
type RulePolicy struct{ roles map[string]*role }

// ReadRulePolicy reads rules from r and creates and returns a new policy or an error.
func ReadRulePolicy(r io.Reader) (*RulePolicy, error) {
	rs, err := ReadRules(r)
	if err != nil {
		return nil, err
	}
	p := &RulePolicy{}
	err = p.Add(rs...)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Police permits role to execute the given actions or returns an error.
func (p *RulePolicy) Police(role string, acts ...Action) error {
	r := p.roles[role]
	if r == nil {
		r = p.roles["*"]
	}
	if r == nil {
		return fmt.Errorf("no permissions for %q", role)
	}
	for _, a := range acts {
		if !r.permitted(a) {
			return fmt.Errorf("no permission %s on %s for %q", a.Op, a.Top, role)
		}
		if r.denied(a) {
			return fmt.Errorf("denied permission %s on %s for %q", a.Op, a.Top, role)
		}
	}
	return nil
}

// Add adds rules to the policy or returns an error.
func (p *RulePolicy) Add(rs ...Rule) error {
	if p.roles == nil {
		p.roles = make(map[string]*role)
	}
	for _, r := range rs {
		if !valid(r) {
			return fmt.Errorf("invalid rule %s", r)
		}
		role := p.role(r.Role)
		if r.Perm == 0 { // add group
			if !hasRole(role.roles, r.Top) {
				role.roles = append(role.roles, p.role(r.Top))
			}
		} else { // add permit or deny
			act := Action{r.Op(), r.Top}
			if r.Perm < 0 {
				role.deny = addAct(role.deny, act)
			} else {
				role.permit = addAct(role.permit, act)
			}
		}
	}
	return nil
}

func addAct(acts []Action, act Action) []Action {
	for i, a := range acts {
		if a.Top != act.Top {
			continue
		}
		acts[i].Op = a.Op | act.Op
		return acts
	}
	return append(acts, act)
}

func hasRole(roles []*role, name string) bool {
	for _, r := range roles {
		if r.name == name {
			return true
		}
	}
	return false
}

func (p *RulePolicy) role(name string) *role {
	r := p.roles[name]
	if r == nil {
		r = &role{name: name}
		if name != "*" {
			r.roles = append(r.roles, p.role("*"))
		}
		p.roles[name] = r
	}
	return r
}

func valid(r Rule) bool { return r.Perm >= -15 && r.Perm <= 15 && r.Top != "" && r.Role != "" }

type role struct {
	name   string
	roles  []*role
	permit []Action
	deny   []Action
}

func (r *role) permitted(act Action) bool {
	for _, a := range r.permit {
		if a.Top == "*" || a.Top == act.Top {
			return a.Op&act.Op == act.Op
		}
	}
	for _, p := range r.roles {
		if p.permitted(act) {
			return true
		}
	}
	return false
}

func (r *role) denied(act Action) bool {
	for _, a := range r.deny {
		if a.Top == "*" || a.Top == act.Top {
			return a.Op&act.Op != 0
		}
	}
	for _, g := range r.roles {
		if g.denied(act) {
			return true
		}
	}
	return false
}
