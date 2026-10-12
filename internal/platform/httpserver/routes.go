package httpserver

import "github.com/gofiber/fiber/v3"

type healthPayload struct {
	Status string `json:"status"`
}

func registerAPI(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", health)
}

func health(c fiber.Ctx) error {
	return c.JSON(healthPayload{Status: "ok"})
}
