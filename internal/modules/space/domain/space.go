package domain

import "strings"

// Scope is a permission a space understands, with consent-page wording.
type Scope struct {
	ID          string
	Description string
}

// Tool is a capability a space exposes, shown in the app's settings. Group
// clusters tools of similar risk and DefaultOn says whether it starts enabled.
type Tool struct {
	Name        string
	Title       string
	Description string
	Scope       string
	Group       string
	ReadOnly    bool
	DefaultOn   bool
}

// Space is one MCP endpoint (for example "coding") served at /api/<slug>.
type Space struct {
	Slug string
	Name string
	// Entry is the scope needed to reach the endpoint at all; each tool then
	// checks its own scope.
	Entry  string
	Scopes []Scope
	Tools  []Tool
}

// FindTool returns the named tool.
func (s Space) FindTool(name string) (Tool, bool) {
	for _, tool := range s.Tools {
		if tool.Name == name {
			return tool, true
		}
	}

	return Tool{}, false
}

// Path is the URL path the MCP endpoint is served on.
func (s Space) Path() string {
	return "/api/" + s.Slug
}

// ResourceURI is the canonical resource identifier (RFC 8707/9728) clients put in `resource`.
func (s Space) ResourceURI(publicURL string) string {
	return strings.TrimRight(publicURL, "/") + s.Path()
}

// MetadataPath is where its Protected Resource Metadata lives (RFC 9728 §3.1, path inserted).
func (s Space) MetadataPath() string {
	return "/.well-known/oauth-protected-resource" + s.Path()
}

// ScopeIDs lists the scope identifiers.
func (s Space) ScopeIDs() []string {
	ids := make([]string, 0, len(s.Scopes))
	for _, scope := range s.Scopes {
		ids = append(ids, scope.ID)
	}

	return ids
}
