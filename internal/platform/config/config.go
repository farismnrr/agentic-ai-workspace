package config

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultWorkers         = 2
	defaultSandboxes       = 4
	defaultSandboxMemoryMB = 2048
	defaultSandboxCPUs     = 1
	defaultPort            = "8081"
	defaultDataDir         = "data"
	defaultSandboxImage    = "node:24-bookworm"
)

// Config contains runtime process configuration.
type Config struct {
	Port              string
	PublicURL         string
	ConnectorURL      string
	DataDir           string
	WorkspacesDir     string
	ClientIPHeader    string
	SandboxImage      string
	SandboxRuntime    string
	SandboxDataVolume string
	AllowedOrigins    []string
	CORSOrigins       []string
	AgentWorkers      int
	MaxSandboxes      int
	SandboxMemoryMB   int
	SandboxCPUs       int
	CookieSecure      bool
}

// Load reads runtime configuration from the environment.
func Load() Config {
	port := getenv("PORT", defaultPort)
	publicURL := strings.TrimRight(getenv("PUBLIC_URL", "http://localhost:"+port), "/")

	connector := strings.TrimRight(strings.TrimSpace(os.Getenv("CONNECTOR_URL")), "/")

	local := []string{"http://localhost:" + port, "http://127.0.0.1:" + port}
	allowed := list("ALLOWED_ORIGINS", local)

	// People sign in and approve connections on the connector's own address, so its origin is allowed too.
	if connector != "" && !slices.Contains(allowed, connector) {
		allowed = append(allowed, connector)
	}

	dataDir := getenv("DATA_DIR", defaultDataDir)

	return Config{
		Port:              port,
		PublicURL:         publicURL,
		ConnectorURL:      connector,
		DataDir:           dataDir,
		WorkspacesDir:     filepath.Join(dataDir, "workspaces"),
		AllowedOrigins:    allowed,
		CORSOrigins:       list("CORS_ORIGINS", allowed),
		ClientIPHeader:    strings.TrimSpace(os.Getenv("CLIENT_IP_HEADER")),
		AgentWorkers:      atoi(getenv("AGENT_WORKERS", strconv.Itoa(defaultWorkers)), defaultWorkers),
		MaxSandboxes:      atoi(getenv("MAX_SANDBOXES", strconv.Itoa(defaultSandboxes)), defaultSandboxes),
		SandboxImage:      getenv("SANDBOX_IMAGE", defaultSandboxImage),
		SandboxRuntime:    strings.TrimSpace(os.Getenv("SANDBOX_RUNTIME")),
		SandboxDataVolume: strings.TrimSpace(os.Getenv("SANDBOX_DATA_VOLUME")),
		SandboxMemoryMB:   atoi(getenv("SANDBOX_MEMORY_MB", strconv.Itoa(defaultSandboxMemoryMB)), defaultSandboxMemoryMB),
		SandboxCPUs:       atoi(getenv("SANDBOX_CPUS", strconv.Itoa(defaultSandboxCPUs)), defaultSandboxCPUs),
		CookieSecure:      strings.HasPrefix(publicURL, "https://") || strings.HasPrefix(connector, "https://"),
	}
}

// Connector is the public address AI apps such as ChatGPT use to reach this server: the root of the
// connector addresses, of the sign-in they are sent through, and of the tokens they are given. It is
// CONNECTOR_URL when set, so the app itself can stay on another address, and PUBLIC_URL otherwise.
func (c Config) Connector() string {
	if c.ConnectorURL != "" {
		return c.ConnectorURL
	}

	return c.PublicURL
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
