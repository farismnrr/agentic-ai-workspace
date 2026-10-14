package httpserver

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
)

// Mount registers one feature's routes on the app. Bootstrap supplies them so
// the platform layer never imports a feature module.
type Mount func(app *fiber.App)

// New builds the outer HTTP adapter and registers all routes. Feature routes
// are registered before the /api not-found guard, and everything before the
// SPA fallback.
func New(frontend fs.FS, mounts ...Mount) *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: bodyLimit})

	registerAPI(app)

	for _, mount := range mounts {
		mount(app)
	}

	app.All("/api/*", apiNotFound)
	registerSPA(app, frontend)

	return app
}
