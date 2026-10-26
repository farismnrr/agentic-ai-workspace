// Package http exposes person-only egress grant approval routes.
package http

import (
	"context"
	"errors"
	"fmt"
	"time"

	accountdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
	egressapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

type Service interface {
	Ask(context.Context, string, string, string, string, domain.Actor) (domain.Grant, bool, error)
	Approve(context.Context, string, domain.Decider, time.Duration) (domain.Grant, error)
	Deny(context.Context, string, domain.Decider) (domain.Grant, error)
	Revoke(context.Context, string, domain.Decider) (domain.Grant, error)
	Get(context.Context, string) (domain.Grant, error)
	ListProject(context.Context, string) ([]domain.Grant, error)
}
type Owner interface {
	Owner(context.Context, string) (string, string, error)
}
type Roles interface {
	Profile(context.Context, string) (accountdomain.User, error)
}
type Handler struct {
	service  Service
	projects Owner
	roles    Roles
	csrf     *CSRF
}

func NewHandler(service Service, projects Owner, roles Roles, csrf *CSRF) *Handler {
	return &Handler{service: service, projects: projects, roles: roles, csrf: csrf}
}

func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		api := app.Group("/api/projects/:id/egress-grants", auth, h.access)
		api.Get("/csrf", h.csrfToken)
		api.Get("/", h.list)
		api.Get("/:grant", h.get)
		const requestsPerMinute = 30
		limit := httpserver.Limit(requestsPerMinute, time.Minute, httpserver.UserID, false)
		api.Post("/", sameOrigin, h.csrf.Validate, limit, h.request)
		api.Post("/:grant/approve", sameOrigin, h.csrf.Validate, limit, h.approve)
		api.Post("/:grant/deny", sameOrigin, h.csrf.Validate, limit, h.deny)
		api.Post("/:grant/revoke", sameOrigin, h.csrf.Validate, limit, h.revoke)
	}
}

// access verifies project ownership and account role on every operation.
func (h *Handler) access(c fiber.Ctx) error {
	user := httpserver.UserID(c)
	profile, err := h.roles.Profile(c.Context(), user)
	if err != nil {
		return errorJSON(c, fiber.StatusInternalServerError, "identity_unavailable")
	}
	owner, _, err := h.projects.Owner(c.Context(), c.Params("id"))
	if err != nil {
		return errorJSON(c, fiber.StatusNotFound, "not_found")
	}
	if owner != user && profile.Role != accountdomain.RoleAdmin {
		return errorJSON(c, fiber.StatusNotFound, "not_found")
	}
	httpserver.SetRole(c, profile.Role)
	return c.Next()
}

func (h *Handler) decider(c fiber.Ctx) domain.Decider {
	return domain.Decider{
		ID: httpserver.UserID(c), ProjectID: c.Params("id"),
		Owner: httpserver.Role(c) != accountdomain.RoleAdmin, Admin: httpserver.Role(c) == accountdomain.RoleAdmin,
	}
}

func (h *Handler) scoped(c fiber.Ctx) (domain.Grant, error) {
	grant, err := h.service.Get(c.Context(), c.Params("grant"))
	if err != nil {
		return domain.Grant{}, fmt.Errorf("load grant: %w", err)
	}
	if grant.ProjectID != c.Params("id") {
		return domain.Grant{}, egressapp.ErrNotFound
	}
	return grant, nil
}

func problem(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, egressapp.ErrNotFound):
		return errorJSON(c, fiber.StatusNotFound, "not_found")
	case errors.Is(err, domain.ErrNotAuthorized):
		return errorJSON(c, fiber.StatusForbidden, "forbidden")
	case errors.Is(err, egressapp.ErrScope), errors.Is(err, domain.ErrNotPending),
		errors.Is(err, domain.ErrNotApproved), errors.Is(err, egressapp.ErrDuplicate), errors.Is(err, egressapp.ErrConflict):
		return errorJSON(c, fiber.StatusConflict, "grant_conflict")
	case errors.Is(err, egressapp.ErrRateLimited), errors.Is(err, egressapp.ErrTooManyPending):
		return errorJSON(c, fiber.StatusTooManyRequests, "rate_limited")
	case errors.Is(err, domain.ErrInvalidHost), errors.Is(err, domain.ErrInvalidRequest),
		errors.Is(err, domain.ErrTTLTooLong):
		return errorJSON(c, fiber.StatusBadRequest, "invalid_request")
	default:
		return errorJSON(c, fiber.StatusInternalServerError, "grant_unavailable")
	}
}

func errorJSON(c fiber.Ctx, status int, code string) error {
	err := httpserver.JSONError(c, status, code)
	if err != nil {
		return fmt.Errorf("egress response: %w", err)
	}
	return nil
}
