package httpserver

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
)

// New builds the outer HTTP adapter and registers all routes.
func New(frontend fs.FS) *fiber.App {
	app := fiber.New()

	registerAPI(app)
	registerSPA(app, frontend)

	return app
}
