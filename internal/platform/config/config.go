package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultWorkers = 2
	defaultPort    = "8080"
	defaultDataDir = "data"
)

// Config contains runtime process configuration.
type Config struct {
	Port           string
	PublicURL      string
	DataDir        string
	CodingRoot     string
	ClientIPHeader string
	SMTPAddr       string
	SMTPUser       string
	SMTPPassword   string
	SMTPFrom       string
	AllowedOrigins []string
	CORSOrigins    []string
	AgentWorkers   int
	CookieSecure   bool
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
		ClientIPHeader: strings.TrimSpace(os.Getenv("CLIENT_IP_HEADER")),
		SMTPAddr:       strings.TrimSpace(os.Getenv("SMTP_ADDR")),
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:       strings.TrimSpace(os.Getenv("SMTP_FROM")),
		AgentWorkers:   atoi(getenv("AGENT_WORKERS", strconv.Itoa(defaultWorkers)), defaultWorkers),
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

func atoi(value string, fallback int) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 1 {
		return fallback
	}

	return number
}
