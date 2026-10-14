package config

import (
	"os"
	"strings"
)

const (
	defaultPort    = "8080"
	defaultDataDir = "data"
)

// Config contains runtime process configuration.
type Config struct {
	Port string
	// PublicURL is the externally visible origin (scheme://host[:port]) used as
	// the OAuth issuer and as the base of every MCP resource identifier.
	PublicURL string
	// DataDir holds the SQLite database and the signing key.
	DataDir string
	// CodingRoot is the sandbox directory exposed by the coding MCP space; empty disables the space.
	CodingRoot string
	// AllowedOrigins are the browser origins allowed to send state-changing requests (CSRF).
	// PublicURL is always allowed in addition. Defaults to localhost and 127.0.0.1 on PORT.
	AllowedOrigins []string
	// CORSOrigins are the origins browsers may call the OAuth endpoints from; defaults to AllowedOrigins.
	CORSOrigins []string
	// CookieSecure marks session cookies Secure; it follows PublicURL's scheme.
	CookieSecure bool
}

// Load reads runtime configuration from the environment.
func Load() Config {
	port := getenv("PORT", defaultPort)
	publicURL := strings.TrimRight(getenv("PUBLIC_URL", "http://localhost:"+port), "/")

	local := []string{"http://localhost:" + port, "http://127.0.0.1:" + port}
	allowed := list("ALLOWED_ORIGINS", local)

	return Config{
		Port:           port,
		PublicURL:      publicURL,
		DataDir:        getenv("DATA_DIR", defaultDataDir),
		CodingRoot:     os.Getenv("CODING_WORKSPACE_ROOT"),
		AllowedOrigins: allowed,
		CORSOrigins:    list("CORS_ORIGINS", allowed),
		CookieSecure:   strings.HasPrefix(publicURL, "https://"),
	}
}

// Address returns the HTTP listen address.
func (c Config) Address() string {
	return ":" + c.Port
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// list reads a comma-separated environment variable, trimming spaces and
// trailing slashes; an unset or empty variable yields the fallback.
func list(key string, fallback []string) []string {
	items := []string{}

	for _, item := range strings.Split(os.Getenv(key), ",") {
		if item = strings.TrimRight(strings.TrimSpace(item), "/"); item != "" {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return fallback
	}

	return items
}
