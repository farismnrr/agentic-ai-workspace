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

// WithInitialScopeHint changes only the scope recommendation on an initial
// unauthorized response. Bearer validation still uses the configured entry
// scope, and authenticated insufficient-scope responses are left untouched.
func WithInitialScopeHint(next http.Handler, scopes []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&initialScopeHint{ResponseWriter: w, scopes: strings.Join(scopes, " ")}, r)
	})
}

type initialScopeHint struct {
	http.ResponseWriter
	scopes string
}

func (w *initialScopeHint) WriteHeader(status int) {
	if status == http.StatusUnauthorized {
		if challenge := w.Header().Get(wwwAuthenticate); strings.HasPrefix(challenge, "Bearer ") {
			w.Header().Set(wwwAuthenticate, replaceScopeHint(challenge, w.scopes))
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

// Unwrap lets http.ResponseController reach the wrapped MCP writer.
func (w *initialScopeHint) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func replaceScopeHint(challenge, scopes string) string {
	parts := splitChallenge(challenge)
	for i, part := range parts[1:] {
		key, _, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.TrimSpace(key) == "scope" {
			parts[i+1] = " scope=\"" + scopes + "\""
			return strings.Join(parts, ",")
		}
	}
	return challenge
}

func splitChallenge(challenge string) []string {
	parts := []string{}
	start := 0
	quoted, escaped := false, false
	for i, r := range challenge {
		if escaped {
			escaped = false
			continue
		}
		if quoted && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
		} else if r == ',' && !quoted {
			parts = append(parts, challenge[start:i])
			start = i + 1
		}
	}
	return append(parts, challenge[start:])
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
