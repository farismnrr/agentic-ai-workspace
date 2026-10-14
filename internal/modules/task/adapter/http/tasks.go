package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type taskPayload struct {
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Proof      *domain.Proof     `json:"proof,omitempty"`
	ID         string            `json:"id"`
	ProjectID  string            `json:"projectId"`
	Title      string            `json:"title"`
	Summary    string            `json:"summary"`
	Status     domain.Status     `json:"status"`
	Mode       domain.Mode       `json:"mode"`
	ProviderID string            `json:"providerId"`
	Model      string            `json:"model"`
	Plan       []domain.PlanStep `json:"plan"`
}

func toPayload(t domain.Task) taskPayload {
	return taskPayload{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, Summary: t.Summary, Status: t.Status, Mode: t.Mode,
		ProviderID: t.ProviderID, Model: t.Model, Plan: append([]domain.PlanStep{}, t.Plan...), Proof: t.Proof,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func (h *Handler) list(c fiber.Ctx) error {
	tasks, err := h.service.List(c.Context(), httpserver.UserID(c), c.Query("project"))
	if err != nil {
		return fail(c, err)
	}

	out := make([]taskPayload, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, toPayload(task))
	}

	return c.JSON(out)
}

func (h *Handler) get(c fiber.Ctx) error {
	task, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(task))
}

func (h *Handler) create(c fiber.Ctx) error {
	var body struct {
		ProjectID  string      `json:"projectId"`
		Prompt     string      `json:"prompt"`
		ProviderID string      `json:"providerId"`
		Model      string      `json:"model"`
		Mode       domain.Mode `json:"mode"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	task, err := h.service.Create(c.Context(), httpserver.UserID(c), application.CreateInput{
		ProjectID: body.ProjectID, Prompt: body.Prompt, ProviderID: body.ProviderID, Model: body.Model, Mode: body.Mode,
	})
	if err != nil {
		return fail(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toPayload(task))
}

func (h *Handler) message(c fiber.Ctx) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	return h.done(c, h.service.FollowUp(c.Context(), httpserver.UserID(c), c.Params("id"), body.Text))
}

func (h *Handler) cancel(c fiber.Ctx) error {
	return h.done(c, h.service.Cancel(c.Context(), httpserver.UserID(c), c.Params("id")))
}

func (h *Handler) accept(c fiber.Ctx) error {
	return h.done(c, h.service.Accept(c.Context(), httpserver.UserID(c), c.Params("id")))
}

func (h *Handler) retry(c fiber.Ctx) error {
	return h.done(c, h.service.Retry(c.Context(), httpserver.UserID(c), c.Params("id")))
}

func (h *Handler) done(c fiber.Ctx, err error) error {
	if err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
