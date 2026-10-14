package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type scopePayload struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type consentPayload struct {
	ClientName   string         `json:"clientName"`
	RedirectHost string         `json:"redirectHost"`
	Scopes       []scopePayload `json:"scopes"`
}

type grantPayload struct {
	CreatedAt    time.Time      `json:"createdAt"`
	LastUsedAt   *time.Time     `json:"lastUsedAt"`
	ID           string         `json:"id"`
	ClientName   string         `json:"clientName"`
	RedirectHost string         `json:"redirectHost"`
	Resource     string         `json:"resource"`
	ResourceName string         `json:"resourceName"`
	Scopes       []scopePayload `json:"scopes"`
}

// signedIn returns the session's user id or writes a 401.
func (h *Handler) signedIn(c fiber.Ctx) (string, bool) {
	userID, err := h.auth.UserID(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("unauthenticated"))

		return "", false
	}

	return userID, true
}

func (h *Handler) describe(c fiber.Ctx) error {
	if _, ok := h.signedIn(c); !ok {
		return nil
	}

	view, err := h.service.Describe(c.Context(), c.Params("id"))
	if errors.Is(err, domain.ErrRequestNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("request_expired"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(consentPayload{
		ClientName: view.ClientName, RedirectHost: view.RedirectHost, Scopes: toScopes(view.Scopes),
	})
}

func (h *Handler) decide(c fiber.Ctx) error {
	userID, ok := h.signedIn(c)
	if !ok {
		return nil
	}

	var body struct {
		Approve bool `json:"approve"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	location, err := h.service.Decide(c.Context(), userID, c.Params("id"), body.Approve)
	if errors.Is(err, domain.ErrRequestNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("request_expired"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(fiber.Map{"redirect": location})
}

func (h *Handler) grants(c fiber.Ctx) error {
	userID, ok := h.signedIn(c)
	if !ok {
		return nil
	}

	grants, err := h.service.ListGrants(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	payload := make([]grantPayload, 0, len(grants))
	for _, grant := range grants {
		payload = append(payload, grantPayload{
			ID: grant.ID, ClientName: grant.ClientName, RedirectHost: domain.RedirectHost(grant.RedirectURI),
			Resource: grant.Resource, ResourceName: h.service.ResourceName(grant.Resource),
			Scopes: toScopes(h.service.ScopeInfo(grant.Resource, grant.Scopes)), CreatedAt: grant.CreatedAt,
			LastUsedAt: grant.LastUsedAt,
		})
	}

	return c.JSON(payload)
}

func (h *Handler) revokeGrant(c fiber.Ctx) error {
	userID, ok := h.signedIn(c)
	if !ok {
		return nil
	}

	err := h.service.RevokeGrant(c.Context(), userID, c.Params("id"))
	if errors.Is(err, domain.ErrGrantNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func toScopes(scopes []domain.Scope) []scopePayload {
	payload := make([]scopePayload, 0, len(scopes))
	for _, scope := range scopes {
		payload = append(payload, scopePayload{ID: scope.ID, Description: scope.Description})
	}

	return payload
}
