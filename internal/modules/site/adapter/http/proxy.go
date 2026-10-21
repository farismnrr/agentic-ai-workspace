package http

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"net"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
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

	preview := h.isPreviewHost(c.Hostname(), site.Slug)
	if h.previewDomain != "" && strings.HasSuffix(strings.ToLower(c.Hostname()), "."+h.previewDomain) && !preview {
		return reply(c, fiber.StatusForbidden, "Project hostname mismatch")
	}
	userID := h.userID(c)
	if preview {
		userID = ""
	}
	valid := ticket != "" && h.service.ValidTicket(site.Token, h.holder(c), ticket)
	if preview && !valid {
		if stored := c.Cookies(previewAccessCookie); stored != "" {
			valid = h.service.ValidTicket(site.Token, h.holder(c), stored)
		}
	}

	switch site.Decide(domain.Access{UserID: userID, Page: isPage(c) && !preview, Ticket: valid}) {
	case domain.AskSignIn:
		return signIn(c)
	case domain.Deny:
		return reply(c, fiber.StatusForbidden, "This app is private.")
	case domain.Missing:
		return missing(c)
	case domain.Allow:
	}

	return h.deliver(c, site, ticket, rest, valid, userID == site.UserID, preview)
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
func (h *Handler) deliver(c fiber.Ctx, site domain.Site, ticket, rest string, valid, owner, preview bool) error {
	if c.Get("Service-Worker") != "" {
		return reply(c, fiber.StatusForbidden, "Service workers are not allowed here")
	}

	if preview && c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		if origin := c.Get("Origin"); origin != "" && origin != "https://"+c.Hostname() {
			return reply(c, fiber.StatusForbidden, "Cross-origin request blocked")
		}
	}
	if c.Method() == fiber.MethodOptions && c.Get("Access-Control-Request-Method") != "" {
		if preview {
			return reply(c, fiber.StatusForbidden, "Cross-origin preflight blocked")
		}
		return preflight(c)
	}

	segment := ""
	if site.Visibility == domain.Private {
		switch {
		case valid && ticket != "":
			segment = ticketPrefix + ticket
		case owner && isPage(c):
			segment = ticketPrefix + h.service.Ticket(site.Token, h.holder(c))
		}
	}

	if !preview && owner && h.previewDomain != "" && isPage(c) {
		if site.Visibility == domain.Private && segment == "" {
			segment = ticketPrefix + h.service.Ticket(site.Token, h.holder(c))
		}
		dest := url.URL{Scheme: "https", Host: site.Slug + "." + h.previewDomain, Path: site.BasePath() + segmentPath(segment) + "/" + rest, RawQuery: string(c.Request().URI().QueryString())}
		return c.Redirect().Status(fiber.StatusFound).To(dest.String())
	}
	if h.needsRedirect(c, site, segment, ticket, valid) {
		target := url.URL{
			Path:     site.BasePath() + segmentPath(segment) + "/" + rest,
			RawQuery: string(c.Request().URI().QueryString()),
		}

		return c.Redirect().Status(fiber.StatusFound).To(target.String())
	}

	err := h.forwardPreview(c, site, "/"+rest, site.BasePath()+segmentPath(segment), preview)
	if err == nil && preview && valid && ticket != "" {
		c.Cookie(&fiber.Cookie{Name: previewAccessCookie, Value: ticket, Path: "/", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	}
	return err
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

func (h *Handler) isPreviewHost(host, slug string) bool {
	if h.previewDomain == "" {
		return false
	}
	name := strings.ToLower(host)
	if stripped, _, err := net.SplitHostPort(name); err == nil {
		name = stripped
	}
	return name == strings.ToLower(slug)+"."+h.previewDomain
}

const previewAccessCookie = "__Host-workspace_preview_access"
