package http

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func (h *Handler) setPreviewTicket(c fiber.Ctx, preview, valid bool, ticket string, err error) {
	if err != nil || !preview || !valid || ticket == "" {
		return
	}
	c.Cookie(&fiber.Cookie{
		Name: previewAccessCookie, Value: ticket, Path: "/", Secure: true, HTTPOnly: true, SameSite: "Lax",
	})
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

func (h *Handler) previewAccess(c fiber.Ctx, site domain.Site, ticket string) (bool, string, bool) {
	preview := h.isPreviewHost(c.Hostname(), site.Slug)
	userID := h.userID(c)
	if preview {
		userID = ""
	}
	valid := ticket != "" && h.service.ValidTicket(site.Token, h.holder(c), ticket)
	if preview && !valid {
		stored := c.Cookies(previewAccessCookie)
		valid = stored != "" && h.service.ValidTicket(site.Token, h.holder(c), stored)
	}
	return preview, userID, valid
}

func (h *Handler) checkPreviewRequest(c fiber.Ctx, preview bool) error {
	if preview && c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		if origin := c.Get("Origin"); origin != "" && origin != "https://"+c.Hostname() {
			return reply(c, fiber.StatusForbidden, "Cross-origin request blocked")
		}
	}

	return nil
}

func (h *Handler) previewSegment(c fiber.Ctx, site domain.Site, ticket string, valid, owner bool) string {
	if site.Visibility != domain.Private {
		return ""
	}
	if valid && ticket != "" {
		return ticketPrefix + ticket
	}
	if owner && isPage(c) {
		return ticketPrefix + h.service.Ticket(site.Token, h.holder(c))
	}
	return ""
}
