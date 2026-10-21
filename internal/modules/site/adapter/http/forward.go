package http

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// forward sends the request to the app through its socket and writes the answer back,
// filtered. The app sees the path without the /run/<name> prefix, and the links of its pages get it back.
func (h *Handler) forwardPreview(c fiber.Ctx, site domain.Site, rest, prefix string, preview bool) error {
	out, err := relay(c.Context(), relayRequest{
		Socket: h.service.Socket(site.ProjectID), Method: c.Method(), Path: rest,
		Query: string(c.Request().URI().QueryString()), Body: c.Body(), Header: requestHeaders(c),
		Base: site.BasePath(), Prefix: prefix, ClientIP: c.IP(), Preview: preview,
	})
	if err != nil {
		return reply(c, err.status, err.text)
	}

	for name, values := range out.Header {
		for i, value := range values {
			if i == 0 {
				c.Set(name, value)
			} else {
				c.Append(name, value)
			}
		}
	}

	return c.Status(out.Status).Send(out.Body)
}

func requestHeaders(c fiber.Ctx) http.Header {
	h := http.Header{}

	for key, value := range c.Request().Header.All() {
		h.Add(string(key), string(value))
	}

	return h
}

// preflight answers a browser's CORS question itself. The sandboxed page has no login, so
// letting every origin ask is harmless, and the app need not know about CORS.
func preflight(c fiber.Ctx) error {
	setSecure(c)
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Set("Access-Control-Max-Age", "600")

	if asked := c.Get("Access-Control-Request-Headers"); asked != "" {
		c.Set("Access-Control-Allow-Headers", asked)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func reply(c fiber.Ctx, status int, text string) error {
	setSecure(c)

	return c.Status(status).SendString(text)
}

func setSecure(c fiber.Ctx) {
	h := http.Header{}
	secure(h)

	for name := range h {
		c.Set(name, h.Get(name))
	}
}

// ticketPrefix marks the address segment that carries a ticket: /run/<name>/~<ticket>/.
const ticketPrefix = "~"

// splitTicket separates the ticket segment from the rest of the path.
func splitTicket(path string) (ticket, rest string) {
	after, found := strings.CutPrefix(path, ticketPrefix)
	if !found {
		return "", path
	}

	ticket, rest, _ = strings.Cut(after, "/")

	return ticket, rest
}

// isPage tells a page the browser navigates to with GET from what a page loads itself and
// from form posts. A browser that sends no fetch metadata is treated as navigating, which is
// the stricter answer for private sites.
func isPage(c fiber.Ctx) bool {
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return false
	}

	dest, mode := c.Get("Sec-Fetch-Dest"), c.Get("Sec-Fetch-Mode")
	if dest == "" && mode == "" {
		return true
	}

	return mode == "navigate" || dest == "document" || dest == "iframe" || dest == "frame"
}
