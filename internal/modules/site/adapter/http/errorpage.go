package http

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
)

// The app's own error screens: the SPA draws them and the server gives them the matching status.
const (
	forbiddenPage = "/forbidden"
	notFoundPage  = "/not-found"
)

// signIn answers a page request that the person may not open by sending them to the private-page
// screen, whose sign-in button brings them back here afterwards.
func signIn(c fiber.Ctx) error {
	next := url.QueryEscape(string(c.Request().URI().RequestURI()))

	return toPage(c, forbiddenPage+"?next="+next)
}

// missing answers a request for a site that is not there. A page the person opens gets the
// not-found screen; anything the page loads itself keeps a plain 404, never HTML.
func missing(c fiber.Ctx) error {
	if isPage(c) {
		return toPage(c, notFoundPage)
	}

	return reply(c, fiber.StatusNotFound, "Not found")
}

func toPage(c fiber.Ctx, target string) error {
	setSecure(c)

	return c.Redirect().Status(fiber.StatusFound).To(target)
}
