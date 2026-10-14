package httpserver

import "github.com/gofiber/fiber/v3"

const bodyLimit = 1 << 20

type healthPayload struct {
	Status string `json:"status"`
}

type errorPayload struct {
	Error string `json:"error"`
}

func registerAPI(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", health)
}

func health(c fiber.Ctx) error {
	return c.JSON(healthPayload{Status: "ok"})
}

// apiNotFound keeps unknown API paths from falling through to the SPA shell.
func apiNotFound(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(errorPayload{Error: "not_found"})
}
