package config

import (
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultWorkers         = 2
	defaultSandboxes       = 4
	defaultSitesPerUser    = 0
	defaultSitesTotal      = 20
	defaultSandboxMemoryMB = 2048
	defaultSandboxCPUs     = 1
	defaultSandboxSwapMB   = 0
	defaultPort            = "8081"
	defaultDataDir         = "data"
	defaultAdminEmail      = "farismunir2@gmail.com"
	defaultSandboxImage    = "ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest"
)

// defaultEgressHosts are the package registries sandboxed commands may reach.
var defaultEgressHosts = []string{
	"pypi.org", "files.pythonhosted.org", "registry.npmjs.org", "proxy.golang.org", "sum.golang.org",
	"storage.googleapis.com", // Go module proxy redirects ZIP downloads here.
	"crates.io", "index.crates.io", "static.crates.io", "static.rust-lang.org",
	"repo.maven.apache.org", "repo1.maven.org",
}

// Hosts the vendors' own CLIs may reach while a person signs in or works with their plan. Each list is
// reachable only from the container of that vendor's CLI.
var (
	defaultClaudeHosts  = []string{"claude.com", "platform.claude.com", "console.anthropic.com", "api.anthropic.com"}
	defaultChatGPTHosts = []string{"auth.openai.com", "chatgpt.com", "api.openai.com"}
)

// Config contains runtime process configuration.
type Config struct {
	GitHub            GitHub
	PublicURL         string
	SitePreviewDomain string
	ConnectorURL      string
	DataDir           string
	ClientIPHeader    string
	SandboxImage      string
	SandboxRuntime    string
	SandboxDataVolume string
	Port              string
	// SearxngURL is the address of a SearXNG server for web search; empty turns web search off.
	SearxngURL       string
	PostgresAdminURL string
	AdminEmail       string
	CORSOrigins      []string
	AllowedOrigins   []string
	SandboxEgress    []string
	ClaudeEgress     []string
	ChatGPTEgress    []string
	AgentWorkers     int
	MaxSandboxes     int
	SiteMaxPerUser   int
	SiteMaxTotal     int
	SandboxMemoryMB  int
	SandboxCPUs      int
	SandboxSwapMB    int
	CookieSecure     bool
}

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
		AllowedOrigins:    allowed,
		CORSOrigins:       list("CORS_ORIGINS", allowed),
		ClientIPHeader:    strings.TrimSpace(os.Getenv("CLIENT_IP_HEADER")),
		AgentWorkers:      atoi(getenv("AGENT_WORKERS", strconv.Itoa(defaultWorkers)), defaultWorkers),
		MaxSandboxes:      atoi(getenv("MAX_SANDBOXES", strconv.Itoa(defaultSandboxes)), defaultSandboxes),
		SiteMaxPerUser:    atoi(getenv("SITE_MAX_PER_USER", strconv.Itoa(defaultSitesPerUser)), defaultSitesPerUser),
		SitePreviewDomain: strings.ToLower(strings.Trim(strings.TrimSpace(os.Getenv("SITE_PREVIEW_DOMAIN")), ".")),
		SiteMaxTotal:      atoi(getenv("SITE_MAX_TOTAL", strconv.Itoa(defaultSitesTotal)), defaultSitesTotal),
		SearxngURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("SEARXNG_URL")), "/"),
		PostgresAdminURL:  strings.TrimSpace(os.Getenv("POSTGRES_ADMIN_URL")),
		AdminEmail:        strings.ToLower(strings.TrimSpace(getenv("ADMIN_EMAIL", defaultAdminEmail))),
		SandboxImage:      getenv("SANDBOX_IMAGE", defaultSandboxImage),
		SandboxRuntime:    strings.TrimSpace(os.Getenv("SANDBOX_RUNTIME")),
		SandboxDataVolume: strings.TrimSpace(os.Getenv("SANDBOX_DATA_VOLUME")),
		SandboxEgress:     list("SANDBOX_EGRESS_HOSTS", defaultEgressHosts),
		ClaudeEgress:      list("CLAUDE_EGRESS_HOSTS", defaultClaudeHosts),
		ChatGPTEgress:     list("CHATGPT_EGRESS_HOSTS", defaultChatGPTHosts),
		SandboxMemoryMB:   atoi(getenv("SANDBOX_MEMORY_MB", strconv.Itoa(defaultSandboxMemoryMB)), defaultSandboxMemoryMB),
		SandboxCPUs:       atoi(getenv("SANDBOX_CPUS", strconv.Itoa(defaultSandboxCPUs)), defaultSandboxCPUs),
		SandboxSwapMB:     atoi(getenv("SANDBOX_SWAP_MB", strconv.Itoa(defaultSandboxSwapMB)), defaultSandboxSwapMB),
		CookieSecure:      strings.HasPrefix(publicURL, "https://") || strings.HasPrefix(connector, "https://"),
		GitHub: GitHub{
			ClientID:     strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
			AppSlug:      strings.TrimSpace(os.Getenv("GITHUB_APP_SLUG")),
		},
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
