package config

// GitHub is the GitHub App people connect their account through. Empty means the feature is off.
type GitHub struct {
	ClientID     string
	ClientSecret string
	AppSlug      string
}

// Enabled reports whether the GitHub App is configured.
func (g GitHub) Enabled() bool {
	return g.ClientID != "" && g.ClientSecret != ""
}
