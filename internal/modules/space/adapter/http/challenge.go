package http

import (
	"net/http"
	"strings"
)

const wwwAuthenticate = "Www-Authenticate"

// withScopeError makes 403 challenges carry error="insufficient_scope", which
// the MCP authorization spec asks for and clients use to start step-up auth.
func withScopeError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&scopeChallenge{ResponseWriter: w}, r)
	})
}

type scopeChallenge struct {
	http.ResponseWriter
}

func (w *scopeChallenge) WriteHeader(status int) {
	if status == http.StatusForbidden {
		header := w.Header()
		if value := header.Get(wwwAuthenticate); strings.HasPrefix(value, "Bearer ") && !strings.Contains(value, "error=") {
			header.Set(wwwAuthenticate, `Bearer error="insufficient_scope", `+strings.TrimPrefix(value, "Bearer "))
		}
	}

	w.ResponseWriter.WriteHeader(status)
}

// Unwrap lets http.ResponseController reach Flush and friends on the real writer.
func (w *scopeChallenge) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
