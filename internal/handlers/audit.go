package handlers

import (
	"context"
	"net/http"
)

// auditFunc records one action. The server wires it to its audit log; tests
// may leave it nil (no-op).
type auditFunc func(action, user, detail string)

// audit fires the audit hook if one is wired. Nil hooks are a no-op so tests
// that build handlers without an audit logger still work.
func audit(f auditFunc, action, user, detail string) {
	if f != nil {
		f(action, user, detail)
	}
}

type ctxKey int

const userContextKey ctxKey = 0

// WithUser stores the authenticated username in the request context so
// handlers can audit who did what.
func WithUser(r *http.Request, user string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, user))
}

// UserFrom returns the username stored by the auth middleware ("" if absent).
func UserFrom(r *http.Request) string {
	u, _ := r.Context().Value(userContextKey).(string)
	return u
}
