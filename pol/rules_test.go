package pol

import (
	"reflect"
	"strings"
	"testing"
)

func TestRuleTextDecode(t *testing.T) {
	tests := []struct {
		raw  string
		want Rule
	}{
		{"+a * admin", Rule{15, "*", "admin"}},
		{"+a\t*\tadmin", Rule{15, "*", "admin"}},
		{"-drwx topic role", Rule{-15, "topic", "role"}},
		{"@  admin  mb0", Rule{0, "admin", "mb0"}},
		{"-rw foo bar", Rule{-6, "foo", "bar"}},
	}
	for _, test := range tests {
		var r Rule
		err := r.UnmarshalText([]byte(test.raw))
		if err != nil {
			t.Errorf("unmarshal %s error %v", test.raw, err)
			continue
		}
		if r != test.want {
			t.Errorf("test %s want %v got %v", test.raw, test.want, r)
		}
	}
}

var testRules = `
# empty lines and comment lines starting with # are ignored

# we can declare role associations before or after other rules
@   admin         mb0
+a  *             admin
+x  self.profile  user
-d  evt.event     *
# you probably only need general roles like admin and user and not add every user name as a role
# but if your use case requires fine-grained per user permissions it can very well be done 
@   user          bob
+x  isyouruncle   bob
+rw isyouruncle   bob
`

func TestRulePolicy(t *testing.T) {
	rs, err := ReadRules(strings.NewReader(testRules))
	if err != nil {
		t.Fatalf("read rules failed: %v", err)
	}
	want := []Rule{
		{0, "admin", "mb0"},
		{15, "*", "admin"},
		{1, "self.profile", "user"},
		{-8, "evt.event", "*"},
		{0, "user", "bob"},
		{1, "isyouruncle", "bob"},
		{6, "isyouruncle", "bob"},
	}
	if !reflect.DeepEqual(rs, want) {
		t.Fatalf("read rules want %v got %v", want, rs)
	}
	var p RulePolicy
	err = p.Add(rs...)
	if err != nil {
		t.Fatalf("failed to add rules: %v", err)
	}

	tests := []struct {
		role string
		acts []Action
		ok   bool
	}{
		{"mb0", []Action{{R, "evt.event"}}, true},
		{"mb0", []Action{{D, "evt.event"}}, false},
		{"mb0", []Action{{D, "prod.prod"}}, true},
		{"bob", []Action{{R, "evt.event"}}, false},
		{"bob", []Action{{X, "self.profile"}}, true},
		{"bob", []Action{{R | W, "self.profile"}}, true},
	}
	for _, test := range tests {
		err := p.Police(test.role, test.acts...)
		if err != nil {
			if test.ok {
				t.Errorf("test %s %v want permit got deny: %v", test.role, test.acts, err)
			}
		} else if !test.ok {
			t.Errorf("test %s %v want deny got permit", test.role, test.acts)
		}
	}
}
