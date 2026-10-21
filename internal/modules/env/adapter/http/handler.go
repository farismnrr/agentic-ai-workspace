package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves project-scoped environment variable endpoints.
type Handler struct{ service *application.Service }

// NewHandler builds the Env HTTP adapter.
func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

// Mount registers owner-only list, replace and delete routes.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/projects/:id/env", auth, h.list)
		app.Put("/api/projects/:id/env", auth, sameOrigin, h.put)
		app.Delete("/api/projects/:id/env/:name", auth, sameOrigin, h.remove)
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
