package httpserver

import "github.com/gofiber/fiber/v3"

// bodyLimit keeps ordinary API request bodies small. Authenticated uploads get
// a larger, route-specific limit from HeaderReceived.
const bodyLimit = 11 << 20

type healthPayload struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type errorPayload struct {
	Error string `json:"error"`
}

func registerAPI(app *fiber.App, version string) {
	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(healthPayload{Status: "ok", Version: version})
	})
}

// stampVersion names the running build on every response, the SPA shell at / included.
func stampVersion(version string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-App-Version", version)

		return c.Next()
	}
}

// apiNotFound keeps unknown API paths from falling through to the SPA shell.
func apiNotFound(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(errorPayload{Error: "not_found"})
}
