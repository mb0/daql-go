package ott

import (
	"testing"
	"time"
)

func TestTokenGenerator(t *testing.T) {
	now := time.Now()
	g := New(Pepper("appSalt", "envPepper"), 3)
	g.Today = func() time.Time {
		return now
	}
	tok := g.Generate("id", "password")
	if !g.Validate(tok, "id", "password") {
		t.Error("same data, time does not validate")
	}
	if g.Validate(tok, "id", "newpassword") {
		t.Error("new password, same time does validate")
	}
	if g.Validate(tok, "otherid", "password") {
		t.Error("other id, same time does validate")
	}
	g.Today = func() time.Time {
		return now.Add(3 * 24 * time.Hour)
	}
	if !g.Validate(tok, "id", "password") {
		t.Error("same data, new time does not validate")
	}
	g.Today = func() time.Time {
		return now.Add(4 * 24 * time.Hour)
	}
	if g.Validate(tok, "id", "password") {
		t.Error("same data, but timeout does validate")
	}
}
