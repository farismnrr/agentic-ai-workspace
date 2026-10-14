package httpserver

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

type userKey struct{}

// Authenticator resolves a browser session cookie to a user id.
type Authenticator interface {
	UserID(ctx context.Context, sessionToken string) (string, error)
}

// RequireSession rejects requests without a valid session and otherwise
// stores the user id for UserID.
func RequireSession(auth Authenticator) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := auth.UserID(c.Context(), c.Cookies(SessionCookieName))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthenticated")
		}

		fiber.StoreInContext(c, userKey{}, userID)

		return c.Next()
	}
}

// UserID returns the signed-in user's id inside a RequireSession route.
func UserID(c fiber.Ctx) string {
	id, _ := fiber.ValueFromContext[string](c, userKey{})

	return id
}
