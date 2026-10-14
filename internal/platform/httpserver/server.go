package httpserver

import (
	"io/fs"

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

	registerAPI(app)

	for _, mount := range mounts {
		mount(app)
	}

	app.All("/api/*", apiNotFound)
	registerSPA(app, frontend)

	return app
}
