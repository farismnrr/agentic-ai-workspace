package http

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// serve answers one request under /run/<name>/. Every answer, errors included,
// carries the sandbox headers, and nothing is forwarded until the request is allowed.
func (h *Handler) serve(c fiber.Ctx) error {
	site, err := h.service.Resolve(c.Context(), c.Params("name"))
	if err != nil {
		return h.old(c)
	}

	ticket, rest := "", c.Params("*")
	if site.Visibility == domain.Private {
		ticket, rest = splitTicket(rest)
	}

	userID := h.userID(c)
	valid := ticket != "" && h.service.ValidTicket(site.Token, h.holder(c), ticket)

	switch site.Decide(domain.Access{UserID: userID, Page: isPage(c), Ticket: valid}) {
	case domain.AskSignIn:
		return signIn(c)
	case domain.Deny:
		return reply(c, fiber.StatusForbidden, "This app is private.")
	case domain.Missing:
		return missing(c)
	case domain.Allow:
	}

	return h.deliver(c, site, ticket, rest, valid, userID == site.UserID)
}

// old sends an address made before names were unique, /run/<name>/<uuid>/..., to the site's
// current one. Anything else under /run that matches no site is missing.
func (h *Handler) old(c fiber.Ctx) error {
	token, rest, _ := strings.Cut(c.Params("*"), "/")

	site, err := h.service.ResolveOld(c.Context(), token)
	if err != nil {
		return missing(c)
	}

	target := url.URL{Path: site.BasePath() + "/" + rest, RawQuery: string(c.Request().URI().QueryString())}

	return c.Redirect().Status(fiber.StatusFound).To(target.String())
}

// deliver answers an allowed request: it redirects to the canonical address, which for a
// private site carries a fresh ticket when the owner opens it, or forwards it to the app.
func (h *Handler) deliver(c fiber.Ctx, site domain.Site, ticket, rest string, valid, owner bool) error {
	if c.Get("Service-Worker") != "" {
		return reply(c, fiber.StatusForbidden, "Service workers are not allowed here")
	}

	if c.Method() == fiber.MethodOptions && c.Get("Access-Control-Request-Method") != "" {
		return preflight(c)
	}

	segment := ""
	if site.Visibility == domain.Private {
		switch {
		case valid:
			segment = ticketPrefix + ticket
		case owner && isPage(c):
			segment = ticketPrefix + h.service.Ticket(site.Token, h.holder(c))
		}
	}

	if h.needsRedirect(c, site, segment, ticket, valid) {
		target := url.URL{
			Path:     site.BasePath() + segmentPath(segment) + "/" + rest,
			RawQuery: string(c.Request().URI().QueryString()),
		}

		return c.Redirect().Status(fiber.StatusFound).To(target.String())
	}

	return h.forward(c, site, "/"+rest, site.BasePath()+segmentPath(segment))
}

func segmentPath(segment string) string {
	if segment == "" {
		return ""
	}

	return "/" + segment
}

// needsRedirect reports whether the address is not the canonical one: the folder must end in
// a slash for relative links to work, and a private page opened by its owner without a
// (valid) ticket must be reopened with one.
func (h *Handler) needsRedirect(c fiber.Ctx, site domain.Site, segment, ticket string, valid bool) bool {
	path := string(c.Request().URI().PathOriginal())
	bare := strings.HasSuffix(path, "/"+site.Slug) || (valid && strings.HasSuffix(path, "/"+ticketPrefix+ticket))

	return bare || (segment != "" && !valid)
}

// holder identifies who a ticket is for: their browser, and their address when it can be trusted.
func (h *Handler) holder(c fiber.Ctx) string {
	if h.bindIP {
		return c.IP() + "|" + c.Get("User-Agent")
	}

	return c.Get("User-Agent")
}

func (h *Handler) userID(c fiber.Ctx) string {
	id, err := h.auth.UserID(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		return ""
	}

	return id
}
