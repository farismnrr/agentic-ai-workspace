package http

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/runtime"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	requestTimeout = 5 * time.Minute
	maxResponse    = 64 << 20
)

// forward sends the request to the app through its socket and writes the answer back,
// filtered. The app sees the path without the /run/<name>/<token> prefix.
func (h *Handler) forward(c fiber.Ctx, site domain.Site, rest, prefix string) error {
	target := url.URL{Scheme: "http", Host: "site", Path: rest, RawQuery: string(c.Request().URI().QueryString())}

	req, err := http.NewRequestWithContext(c.Context(), c.Method(), target.String(), bytes.NewReader(c.Body()))
	if err != nil {
		return reply(c, fiber.StatusBadRequest, "Bad request")
	}

	req.Header = forRequest(requestHeaders(c), prefix, c.IP())
	req.Host = "site"

	resp, err := runtime.Client(h.service.Socket(site.ProjectID), requestTimeout).Do(req)
	if err != nil {
		return reply(c, fiber.StatusBadGateway, "The app is not answering.")
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		return reply(c, fiber.StatusBadGateway, "The app's answer was too large.")
	}

	for name, values := range forResponse(resp.Header, prefix) {
		for i, value := range values {
			if i == 0 {
				c.Set(name, value)
			} else {
				c.Append(name, value)
			}
		}
	}

	return c.Status(resp.StatusCode).Send(body)
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

// ticketPrefix marks the address segment that carries a ticket: /run/<name>/<token>/~<ticket>/.
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
