package httpserver

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	staticmiddleware "github.com/gofiber/fiber/v3/middleware/static"
)

func registerSPA(app *fiber.App, frontend fs.FS) {
	app.Use("/", staticmiddleware.New("", staticmiddleware.Config{
		FS: frontend,
	}))
	app.Get("/*", spaFallback(frontend))
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
