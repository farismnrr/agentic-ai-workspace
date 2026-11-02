package httpserver

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// Mount registers one feature's routes on the app. Bootstrap supplies them so
// the platform layer never imports a feature module.
type Mount func(app *fiber.App)

// Settings are the process-wide HTTP options.
type Settings struct {
	// ClientIPHeader makes c.IP() read the real client address from this header
	// when the request comes from a local or private proxy.
	ClientIPHeader string
	// Version is the build's version, injected with -ldflags; empty means "dev".
	Version string
}

// New builds the outer HTTP adapter and registers all routes. Feature routes
// are registered before the /api not-found guard, and everything before the
// SPA fallback.
func New(frontend fs.FS, settings Settings, mounts ...Mount) *fiber.App {
	config := fiber.Config{BodyLimit: bodyLimit}
	if settings.ClientIPHeader != "" {
		config.ProxyHeader = settings.ClientIPHeader
		config.TrustProxy = true
		config.TrustProxyConfig = fiber.TrustProxyConfig{Loopback: true, Private: true}
	}

	app := fiber.New(config)
	app.Use(noStoreEnvReveal)

	version := settings.Version
	if version == "" {
		version = "dev"
	}

	app.Use(stampVersion(version))
	registerAPI(app, version)

	for _, mount := range mounts {
		mount(app)
	}

	app.All("/api/*", apiNotFound)
	registerSPA(app, frontend)

	return app
}

func noStoreEnvReveal(c fiber.Ctx) error {
	if !isEnvRevealPath(c.Path()) {
		return c.Next()
	}

	setEnvRevealNoStore(c)
	err := c.Next()
	setEnvRevealNoStore(c)
	return err
}

func isEnvRevealPath(path string) bool {
	if !strings.HasPrefix(path, "/api/projects/") || !strings.HasSuffix(path, "/reveal") {
		return false
	}

	segments := strings.Split(path, "/")
	return len(segments) == 7 && segments[1] == "api" && segments[2] == "projects" &&
		segments[3] != "" && segments[4] == "env" && segments[5] != "" && segments[6] == "reveal"
}

func setEnvRevealNoStore(c fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("Pragma", "no-cache")
}
