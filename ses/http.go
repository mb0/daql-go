package ses

import (
	"context"
	"net/http"
)

// ContextKey is the http request context key for a session pointer.
const ContextKey sesKey = "ses"

type sesKey string

// Decorate is the default request context decorator used by manager.
func Decorate(r *http.Request, s *Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ContextKey, s))
}

// Ger returns the session from a http request context or nil.
func Get(r *http.Request) *Session {
	s, _ := r.Context().Value(ContextKey).(*Session)
	return s
}

// Provide returns adds a http middleware that injects a session provider.
func Provide(m *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Provider{m, next}
	}
}

// Provider is a http handler that provides sessions to all handled http requests.
type Provider struct {
	*Manager
	Next http.Handler
}

func (m Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only provide session if no previous session was authenticated
	s, ok := r.Context().Value(ContextKey).(*Session)
	if !ok || s.User() == "" {
		r = m.ReadOrCreate(w, r, true)
	}
	m.Next.ServeHTTP(w, r)
}

// Requirer is a http handler that checks sessions before proceeding with the next handler.
type Requirer struct {
	// Allow returns whether the request is allowed to proceed. On failure Allow should write an
	// error or redirect to the response writer. There are usually two error classes:
	// unauthenticated and unauthorized requests. Unauthenticated request should be redirected
	// to a login page, unauthorized requests should see a simple error page with a back link.
	Allow func(http.ResponseWriter, *http.Request, *Session) bool
	Next  http.Handler
}

func (c Requirer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.Allow(w, r, Get(r)) {
		c.Next.ServeHTTP(w, r)
	}
}
