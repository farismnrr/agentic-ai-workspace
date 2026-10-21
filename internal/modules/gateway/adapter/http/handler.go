// Package http serves the endpoints that let a project owner choose a model gateway.
package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves /api/projects/:id/gateway.
type Handler struct{ service *application.Service }

// NewHandler builds the adapter.
func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

// Mount registers the routes.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/projects/:id/gateway", auth, h.get)
		app.Put("/api/projects/:id/gateway", auth, sameOrigin, h.put)
		app.Delete("/api/projects/:id/gateway", auth, sameOrigin, h.remove)
	}
}

func (h *Handler) get(c fiber.Ctx) error {
	view, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	if view.Options == nil {
		view.Options = []domain.Choice{}
	}

	return c.JSON(view)
}

func (h *Handler) put(c fiber.Ctx) error {
	var body struct {
		ProviderID string `json:"providerId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.service.Set(c.Context(), httpserver.UserID(c), c.Params("id"), body.ProviderID); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Clear(c.Context(), httpserver.UserID(c), c.Params("id")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	case errors.Is(err, domain.ErrIneligible):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("provider_not_allowed"))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
}
