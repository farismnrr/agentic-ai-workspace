package server

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	staticmiddleware "github.com/gofiber/fiber/v3/middleware/static"
)

func New(frontend fs.FS) *fiber.App {
	app := fiber.New()

	registerAPI(app)

	app.Use("/", staticmiddleware.New("", staticmiddleware.Config{
		FS: frontend,
	}))

	app.Get("/*", spaFallback(frontend))

	return app
}

func registerAPI(app *fiber.App) {
	api := app.Group("/api")

	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}

func spaFallback(frontend fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		index, err := fs.ReadFile(frontend, "index.html")
		if err != nil {
			return fiber.ErrInternalServerError
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(index)
	}
}
