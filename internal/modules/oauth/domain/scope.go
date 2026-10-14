package domain

import "strings"

// OpenID Connect scopes every authorization server request may use.
const (
	ScopeOpenID        = "openid"
	ScopeEmail         = "email"
	ScopeProfile       = "profile"
	ScopeOfflineAccess = "offline_access"
)

// Scope is a permission with the plain-language text shown on the consent page.
type Scope struct {
	ID          string
	Description string
}

// IdentityScopes are the OIDC scopes, independent of any protected resource.
var IdentityScopes = []Scope{
	{ID: ScopeOpenID, Description: "Know that it is you signing in"},
	{ID: ScopeEmail, Description: "See your email address"},
	{ID: ScopeProfile, Description: "See your name"},
	{ID: ScopeOfflineAccess, Description: "Stay connected until you remove access"},
}

// SplitScopes parses a space-delimited scope string, dropping duplicates.
func SplitScopes(raw string) []string {
	seen := map[string]bool{}
	scopes := []string{}

	for _, scope := range strings.Fields(raw) {
		if !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

// JoinScopes renders scopes as a space-delimited string.
func JoinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

// HasScope reports whether scopes contains want.
func HasScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}

	return false
}
