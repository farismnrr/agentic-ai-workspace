package http

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves project-scoped environment variable endpoints.
type Handler struct {
	service  *application.Service
	verifier passwordVerifier
	audit    revealAudit
}

type passwordVerifier interface {
	VerifyPassword(ctx context.Context, userID, password string) (bool, error)
}

type revealAudit interface {
	RecordReveal(ctx context.Context, userID, projectID, name, outcome string, at time.Time) error
}

// NewHandler builds the Env HTTP adapter.
func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) WithReveal(verifier passwordVerifier, audit revealAudit) *Handler {
	h.verifier, h.audit = verifier, audit
	return h
}

// Mount registers owner-only list, replace and delete routes.
func (h *Handler) Mount(auth, sameOrigin, revealLimit fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/projects/:id/env", auth, h.list)
		app.Put("/api/projects/:id/env", auth, sameOrigin, h.put)
		app.Delete("/api/projects/:id/env/:name", auth, sameOrigin, h.remove)
		revealPath := "/api/projects/:id/env/:name/reveal"
		app.Post(revealPath, auth, sameOrigin, revealLimit, h.reveal)
	}
}

type payload struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Value     *string   `json:"value,omitempty"`
	Name      string    `json:"name"`
	IsSecret  bool      `json:"isSecret"`
}

type input struct {
	IsSecret *bool  `json:"isSecret"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

type revealInput struct {
	Password string `json:"password"`
}

type revealPayload struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h *Handler) reveal(c fiber.Ctx) error {
	if len(c.Body()) > 2048 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(httpserver.Failure("invalid_request"))
	}
	var body revealInput
	if err := c.Bind().JSON(&body); err != nil || body.Password == "" || h.verifier == nil || h.audit == nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_request"))
	}
	userID, projectID, name := httpserver.UserID(c), c.Params("id"), c.Params("name")
	if err := h.service.AuthorizeReveal(c.Context(), userID, projectID); err != nil {
		_ = h.audit.RecordReveal(c.Context(), userID, projectID, "", "denied", time.Now())
		return fail(c, err)
	}
	valid, err := h.verifier.VerifyPassword(c.Context(), userID, body.Password)
	body.Password = ""
	if err != nil {
		_ = h.audit.RecordReveal(c.Context(), userID, projectID, name, "unavailable", time.Now())
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	if !valid {
		if auditErr := h.audit.RecordReveal(c.Context(), userID, projectID, name, "password_rejected", time.Now()); auditErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
		}
		return c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("invalid_credentials"))
	}
	value, err := h.service.Secret(c.Context(), userID, projectID, name)
	if err != nil {
		_ = h.audit.RecordReveal(c.Context(), userID, projectID, name, "denied", time.Now())
		return fail(c, err)
	}
	if err := h.audit.RecordReveal(c.Context(), userID, projectID, name, "success", time.Now()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	return c.JSON(revealPayload{Name: name, Value: value})
}

func (h *Handler) list(c fiber.Ctx) error {
	variables, err := h.service.List(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	out := make([]payload, 0, len(variables))
	for _, variable := range variables {
		item := payload{Name: variable.Name, IsSecret: variable.IsSecret, UpdatedAt: variable.UpdatedAt}
		if !variable.IsSecret {
			value := variable.Value
			item.Value = &value
		}
		out = append(out, item)
	}

	return c.JSON(out)
}

func (h *Handler) put(c fiber.Ctx) error {
	var body input
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}
	isSecret := true
	if body.IsSecret != nil {
		isSecret = *body.IsSecret
	}

	variable, err := h.service.Put(c.Context(), httpserver.UserID(c), c.Params("id"),
		body.Name, body.Value, isSecret)
	if err != nil {
		return fail(c, err)
	}
	item := payload{Name: variable.Name, IsSecret: variable.IsSecret, UpdatedAt: variable.UpdatedAt}
	if !variable.IsSecret {
		value := variable.Value
		item.Value = &value
	}

	return c.JSON(item)
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Delete(c.Context(), httpserver.UserID(c), c.Params("id"), c.Params("name")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	case errors.Is(err, domain.ErrInvalidName):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_name"))
	case errors.Is(err, domain.ErrReserved):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("reserved_name"))
	case errors.Is(err, domain.ErrInvalidValue):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_value"))
	case errors.Is(err, domain.ErrLimit):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("variable_limit"))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
}
