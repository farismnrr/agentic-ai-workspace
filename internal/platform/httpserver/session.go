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
// stores the user id for UserID. It also refuses requests the browser marks as coming
// from another site (see crossSite), so a page running in a sandbox cannot use the
// person's session even if it learns an API address.
func RequireSession(auth Authenticator) fiber.Handler {
	return func(c fiber.Ctx) error {
		if crossSite(c) {
			return JSONError(c, fiber.StatusForbidden, "cross_site")
		}

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

// crossSite follows web.dev's resource isolation policy: a request the browser labels
// Sec-Fetch-Site: cross-site is refused unless it is a top-level navigation, such as someone
// following a link to the app. Requests from a sandboxed page, which has an opaque origin,
// are cross-site. Browsers that send no metadata are not affected.
func crossSite(c fiber.Ctx) bool {
	if c.Get("Sec-Fetch-Site") != "cross-site" {
		return false
	}

	navigation := c.Method() == fiber.MethodGet &&
		c.Get("Sec-Fetch-Mode") == "navigate" && c.Get("Sec-Fetch-Dest") == "document"

	return !navigation
}
