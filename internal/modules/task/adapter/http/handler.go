package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves the task endpoints.
type Handler struct {
	service *application.Service
	now     func() time.Time
}

// NewHandler builds the task HTTP adapter.
func NewHandler(service *application.Service, now func() time.Time) *Handler {
	return &Handler{service: service, now: now}
}

// Mount registers the routes; auth guards all, sameOrigin the state-changing ones.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/tasks", auth, h.list)
		app.Post("/api/tasks", auth, sameOrigin, h.create)
		app.Get("/api/tasks/:id", auth, h.get)
		app.Get("/api/tasks/:id/events", auth, h.events)
		app.Post("/api/tasks/:id/messages", auth, sameOrigin, h.message)
		app.Post("/api/tasks/:id/events/:event/answer", auth, sameOrigin, h.answer)
		app.Post("/api/tasks/:id/events/:event/undo", auth, sameOrigin, h.undo)
		app.Post("/api/tasks/:id/cancel", auth, sameOrigin, h.cancel)
		app.Post("/api/tasks/:id/accept", auth, sameOrigin, h.accept)
		app.Post("/api/tasks/:id/retry", auth, sameOrigin, h.retry)
		app.Get("/api/usage", auth, h.usage)
	}
}

type statusError struct {
	err    error
	code   string
	status int
}

var mapped = []statusError{
	{domain.ErrNotFound, "not_found", fiber.StatusNotFound},
	{domain.ErrBadRequest, "bad_request", fiber.StatusBadRequest},
	{domain.ErrNoProject, "no_project", fiber.StatusBadRequest},
	{domain.ErrNoModel, "no_model", fiber.StatusConflict},
	{domain.ErrBusy, "busy", fiber.StatusConflict},
	{domain.ErrNeedsAnswer, "needs_answer", fiber.StatusConflict},
	{domain.ErrNotWaiting, "not_waiting", fiber.StatusConflict},
	{domain.ErrWrongState, "wrong_state", fiber.StatusConflict},
	{domain.ErrUndoRefused, "undo_refused", fiber.StatusConflict},
}

func fail(c fiber.Ctx, err error) error {
	for _, known := range mapped {
		if errors.Is(err, known.err) {
			return c.Status(known.status).JSON(httpserver.Failure(known.code))
		}
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}

func badBody(c fiber.Ctx) error {
	return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
}
