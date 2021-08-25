// Package ott provides a one-time token implementation, for common account registration tasks.
// The tokens can be used to authorized for password reset or email confirmation link.
package ott

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"time"
)

// Pepper returns the sha256 hash of two values to be used as salt. One value is usually part of the
// application source code and compiled into the binary, the other values is usually read from the
// system environment at program runtime. This way an attacker needs both values to compromise the
// salt. The second good reason is that we can be certain to always have a salt with 32 bytes.
//
// This helper is generally useful but does not justify a separate package.
func Pepper(salt, pepper string) []byte {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(pepper))
	return h.Sum(nil)
}

// Generator is used to generate and validate this variant of one-time tokens.
// Both generate and validate expect data arguments. These arguments should usually include a value
// that would be affected by the action the token authorizes. A password reset token would pass
// in the revision of the credentials or the password itself.
type Generator struct {
	salt        []byte
	TimeoutDays uint16
	// Today returns the current time and can be mocked for testing purpose.
	Today func() time.Time
}

// New returns a new generator with the given salt and timeout in days.
func New(salt []byte, days uint16) *Generator {
	return &Generator{salt, days, time.Now}
}

// Generate returns a new token that is valid for a number of days and the given data.
func (g *Generator) Generate(data ...string) string {
	tok := g.token(days(g.Today()), data)
	return base64.URLEncoding.EncodeToString(tok)
}

// Validate returns whether token is currently valid for the given data.
func (g *Generator) Validate(token string, data ...string) bool {
	b, err := base64.URLEncoding.DecodeString(token)
	if err != nil || len(b) != 30 {
		return false
	}
	ts := uint16(b[0])<<8 | uint16(b[1])
	if days(g.Today())-ts > g.TimeoutDays {
		return false
	}
	return subtle.ConstantTimeCompare(b, g.token(ts, data)) == 1
}

func (g *Generator) token(ts uint16, data []string) []byte {
	b := make([]byte, 2, 30)
	b[0] = byte(ts >> 8)
	b[1] = byte(ts)
	mac := hmac.New(sha256.New224, g.salt)
	for _, d := range data {
		mac.Write([]byte(d))
	}
	mac.Write(b)
	return mac.Sum(b)
}

// 2010-01-01 00:00:00 UTC
const startSec = 1262304000
const daySecs = 24 * 60 * 60

func days(t time.Time) uint16 { return uint16((t.Unix() - startSec) / daySecs) }
