package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves the project endpoints.
type Handler struct {
	service *application.Service
}

// NewHandler builds the project HTTP adapter.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// Mount registers the routes; auth guards all, sameOrigin the state-changing ones.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		api := app.Group("/api/projects", auth)
		api.Get("/", h.list)
		api.Post("/", sameOrigin, h.create)
		api.Get("/id", h.slug)
		api.Get("/:id", h.get)
		api.Put("/:id", sameOrigin, h.update)
		api.Delete("/:id", sameOrigin, h.archive)
		api.Delete("/:id/forever", sameOrigin, h.remove)
		api.Get("/:id/checks", h.checks)
		api.Put("/:id/checks", sameOrigin, h.saveChecks)
	}
}

type payload struct {
	CreatedAt   time.Time `json:"createdAt"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Slug        string    `json:"slug"`
	Source      string    `json:"source"`
}

type input struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Slug is the project ID; only a new project takes it (empty: made from the name) and it never changes.
	Slug string `json:"slug"`
}

func toPayload(p domain.Project) payload {
	return payload{
		ID: p.ID, Name: p.Name, Description: p.Description, Slug: p.Slug, Source: p.Source, CreatedAt: p.CreatedAt,
	}
}

func (h *Handler) list(c fiber.Ctx) error {
	projects, err := h.service.List(c.Context(), httpserver.UserID(c))
	if err != nil {
		return fail(c, err)
	}

	out := make([]payload, 0, len(projects))
	for _, project := range projects {
		out = append(out, toPayload(project))
	}

	return c.JSON(out)
}

func (h *Handler) get(c fiber.Ctx) error {
	project, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(project))
}

// slug answers whether a project ID is valid and free, with the nearest free one when it is not, so the page can
// say so while the person types. Without an id it answers for the one the name would make.
func (h *Handler) slug(c fiber.Ctx) error {
	status, err := h.service.CheckSlug(c.Context(), c.Query("id"), c.Query("name"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(fiber.Map{"valid": status.Valid, "available": status.Available, "suggestion": status.Suggestion})
}

func (h *Handler) create(c fiber.Ctx) error {
	var body input
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	project, err := h.service.Create(c.Context(), httpserver.UserID(c), body.Name, body.Description, body.Slug)
	if err != nil {
		return fail(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toPayload(project))
}

func (h *Handler) update(c fiber.Ctx) error {
	var body input
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	project, err := h.service.Update(c.Context(), httpserver.UserID(c), c.Params("id"), body.Name, body.Description)
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(project))
}

func (h *Handler) archive(c fiber.Ctx) error {
	if err := h.service.Archive(c.Context(), httpserver.UserID(c), c.Params("id")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// remove deletes a project for good, with its files and tasks. The page asks the person first.
func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Delete(c.Context(), httpserver.UserID(c), c.Params("id")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidName):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_name"))
	case errors.Is(err, domain.ErrInvalidDescription):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_description"))
	case errors.Is(err, domain.ErrInvalidChecks):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_checks"))
	case errors.Is(err, domain.ErrInvalidSlug):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_id"))
	case errors.Is(err, domain.ErrSlugTaken):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("id_taken"))
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
