package http

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves the chat endpoints.
type Handler struct {
	service *application.Service
}

// NewHandler builds the chat HTTP adapter.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// Mount registers the routes; auth guards all, sameOrigin the state-changing ones.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/projects/:id/chat", auth, h.get)
		app.Post("/api/projects/:id/chat", auth, sameOrigin, h.send)
		app.Delete("/api/projects/:id/chat", auth, sameOrigin, h.clear)
		app.Post("/api/projects/:id/chat/escalate", auth, sameOrigin, h.escalate)
	}
}

type sendInput struct {
	Prompt string `json:"prompt"`
}

type escalateInput struct {
	Prompt string `json:"prompt"`
	Mode   string `json:"mode"`
}

func (h *Handler) get(c fiber.Ctx) error {
	userID, projectID := httpserver.UserID(c), c.Params("id")
	chat, msgs, err := h.service.Conversation(c.Context(), userID, projectID)
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(map[string]any{
		"chat":     chat,
		"messages": msgs,
	})
}

func (h *Handler) send(c fiber.Ctx) error {
	var body sendInput
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	userID, projectID := httpserver.UserID(c), c.Params("id")
	if strings.Contains(c.Get("Accept"), "text/event-stream") {
		return h.sendStream(c, userID, projectID, body.Prompt)
	}

	userMsg, assistantMsg, err := h.service.Send(c.Context(), userID, projectID, body.Prompt, nil)
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(map[string]any{
		"userMessage":      userMsg,
		"assistantMessage": assistantMsg,
	})
}

func (h *Handler) clear(c fiber.Ctx) error {
	userID, projectID := httpserver.UserID(c), c.Params("id")
	if err := h.service.Clear(c.Context(), userID, projectID); err != nil {
		return fail(c, err)
	}

	return c.JSON(map[string]any{"ok": true})
}

func (h *Handler) escalate(c fiber.Ctx) error {
	var body escalateInput
	_ = c.Bind().JSON(&body)

	userID, projectID := httpserver.UserID(c), c.Params("id")
	taskID, err := h.service.Escalate(c.Context(), userID, application.EscalateInput{
		ProjectID: projectID,
		Prompt:    body.Prompt,
		Mode:      body.Mode,
	})
	if err != nil {
		return fail(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(map[string]string{"taskId": taskID})
}

var mapped = []struct {
	err    error
	code   string
	status int
}{
	{domain.ErrNotFound, "not_found", fiber.StatusNotFound},
	{domain.ErrBadRequest, "bad_request", fiber.StatusBadRequest},
	{domain.ErrNoProject, "no_project", fiber.StatusBadRequest},
	{domain.ErrNoModel, "no_model", fiber.StatusConflict},
}

func fail(c fiber.Ctx, err error) error {
	for _, m := range mapped {
		if errors.Is(err, m.err) {
			return c.Status(m.status).JSON(httpserver.Failure(m.code))
		}
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
