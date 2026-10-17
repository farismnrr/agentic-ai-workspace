package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type eventPayload struct {
	domain.Event
	CanUndo bool `json:"canUndo"`
}

// events returns the thread entries after a cursor, which is how the app follows a running task.
func (h *Handler) events(c fiber.Ctx) error {
	after := int64(fiber.Query[int](c, "after", 0))

	events, err := h.service.Events(c.Context(), httpserver.UserID(c), c.Params("id"), after)
	if err != nil {
		return fail(c, err)
	}

	out := make([]eventPayload, 0, len(events))
	for _, event := range events {
		out = append(out, eventPayload{Event: event, CanUndo: event.CanUndo()})
	}

	return c.JSON(out)
}

func (h *Handler) answer(c fiber.Ctx) error {
	var body struct {
		Allow   *bool      `json:"allow"`
		Choice  string     `json:"choice"`
		Answers [][]string `json:"answers"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	err := h.service.Answer(c.Context(), httpserver.UserID(c), c.Params("id"), c.Params("event"),
		application.Answer{Allow: body.Allow, Choice: body.Choice, Answers: body.Answers})

	return h.done(c, err)
}

func (h *Handler) undo(c fiber.Ctx) error {
	return h.done(c, h.service.UndoChange(c.Context(), httpserver.UserID(c), c.Params("id"), c.Params("event")))
}

type usagePayload struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	Tasks      int    `json:"tasks"`
	Input      int64  `json:"inputTokens"`
	Output     int64  `json:"outputTokens"`
}

// usage sums the person's model usage for the current calendar month.
func (h *Handler) usage(c fiber.Ctx) error {
	now := h.now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	rows, err := h.service.Usage(c.Context(), httpserver.UserID(c), start)
	if err != nil {
		return fail(c, err)
	}

	out := make([]usagePayload, 0, len(rows))
	for _, row := range rows {
		out = append(out, usagePayload{
			ProviderID: row.ProviderID, Model: row.Model, Tasks: row.Tasks, Input: row.Input, Output: row.Output,
		})
	}

	return c.JSON(fiber.Map{"since": start, "models": out})
}

// shot sends one screenshot of a task. It is only an image, so a page may show it with an <img> tag.
func (h *Handler) shot(c fiber.Ctx) error {
	data, err := h.service.Shot(c.Context(), httpserver.UserID(c), c.Params("id"), c.Params("name"))
	if err != nil {
		return fail(c, err)
	}

	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "private, max-age=31536000, immutable")
	c.Set("X-Content-Type-Options", "nosniff")

	return c.Send(data)
}
