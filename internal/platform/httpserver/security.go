package httpserver

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// SessionCookieName is the browser session cookie shared by every feature that
// needs to know who is signed in.
const SessionCookieName = "session"

// SameOrigin rejects state-changing requests from browser origins that are not
// allowed. Browsers always send Origin on cross-site POSTs, so a mismatch is a
// CSRF attempt; requests without Origin (curl, server-to-server) pass. The
// public URL is always allowed in addition to the configured origins.
func SameOrigin(publicURL string, allowed []string) fiber.Handler {
	origins := map[string]bool{strings.ToLower(publicURL): true}
	for _, origin := range allowed {
		origins[strings.ToLower(origin)] = true
	}

	return func(c fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}

		if origin := c.Get(fiber.HeaderOrigin); origin != "" && !origins[strings.ToLower(origin)] {
			return JSONError(c, fiber.StatusForbidden, "cross_origin")
		}

		return c.Next()
	}
}

// CORS lets the listed origins call cookie-less protocol endpoints from a
// browser; "*" opens them to any origin.
func CORS(origins []string) fiber.Handler {
	return cors.New(cors.Config{AllowOrigins: origins})
}

// Failure is the standard error body; handlers send it with c.Status(code).JSON(Failure("reason")).
func Failure(code string) any {
	return errorPayload{Error: code}
}

// JSONError writes the standard error body.
func JSONError(c fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(errorPayload{Error: code})
}
