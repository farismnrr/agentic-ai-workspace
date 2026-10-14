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
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const (
	requestTimeout = 5 * time.Minute
	maxResponse    = 64 << 20
)

// serve answers one request under /run/<name>/<token>/. Every answer, errors included,
// carries the sandbox headers, and nothing is forwarded until the request is allowed.
func (h *Handler) serve(c fiber.Ctx) error {
	site, err := h.service.Resolve(c.Context(), c.Params("token"))
	if err != nil || !site.CanView(h.userID(c), isDocument(c)) {
		return reply(c, fiber.StatusNotFound, "Not found")
	}

	if c.Get("Service-Worker") != "" {
		return reply(c, fiber.StatusForbidden, "Service workers are not allowed here")
	}

	if c.Method() == fiber.MethodOptions && c.Get("Access-Control-Request-Method") != "" {
		return preflight(c)
	}

	rest := "/" + c.Params("*")
	if h.needsRedirect(c, site) {
		query := ""
		if q := string(c.Request().URI().QueryString()); q != "" {
			query = "?" + q
		}

		return c.Redirect().Status(fiber.StatusFound).To(site.BasePath() + rest + query)
	}

	return h.forward(c, site, rest)
}

// needsRedirect reports whether the address must be redirected to the site's own: the readable
// name is only decoration, but the folder must end in a slash for relative links to work.
func (h *Handler) needsRedirect(c fiber.Ctx, site domain.Site) bool {
	path := string(c.Request().URI().PathOriginal())

	return c.Params("name") != site.Slug || strings.HasSuffix(path, "/"+site.Token)
}

func (h *Handler) userID(c fiber.Ctx) string {
	id, err := h.auth.UserID(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		return ""
	}

	return id
}

// forward sends the request to the app through its socket and writes the answer back,
// filtered. The app sees the path without the /run/<name>/<token> prefix.
func (h *Handler) forward(c fiber.Ctx, site domain.Site, rest string) error {
	target := url.URL{Scheme: "http", Host: "site", Path: rest, RawQuery: string(c.Request().URI().QueryString())}

	req, err := http.NewRequestWithContext(c.Context(), c.Method(), target.String(), bytes.NewReader(c.Body()))
	if err != nil {
		return reply(c, fiber.StatusBadRequest, "Bad request")
	}

	req.Header = forRequest(requestHeaders(c), site.BasePath(), c.IP())
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

	for name, values := range forResponse(resp.Header, site.BasePath()) {
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

// isDocument tells a page the browser navigates to from what a page loads itself. A
// browser that sends no fetch metadata is treated as navigating, which is the stricter
// answer for private sites.
func isDocument(c fiber.Ctx) bool {
	dest, mode := c.Get("Sec-Fetch-Dest"), c.Get("Sec-Fetch-Mode")
	if dest == "" && mode == "" {
		return true
	}

	return mode == "navigate" || dest == "document" || dest == "iframe" || dest == "frame"
}
