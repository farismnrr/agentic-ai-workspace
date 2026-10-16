package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type taskPayload struct {
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
	Proof      *domain.Proof `json:"proof,omitempty"`
	ProviderID string        `json:"providerId"`
	ProjectID  string        `json:"projectId"`
	Title      string        `json:"title"`
	Summary    string        `json:"summary"`
	// PRD is the requirements document the team's Planner wrote, in markdown; empty when there is none.
	PRD    string            `json:"prd"`
	Status domain.Status     `json:"status"`
	ID     string            `json:"id"`
	Model  string            `json:"model"`
	Effort string            `json:"effort"`
	Source string            `json:"source"`
	Plan   []domain.PlanStep `json:"plan"`
	// ContextUsed and ContextWindow are tokens; both are zero before the first model call.
	ContextUsed   int64 `json:"contextUsed"`
	ContextWindow int64 `json:"contextWindow"`
	// Team is true when the agent leads a team of teammates on this task.
	Team bool `json:"team"`
}

func toPayload(t domain.Task) taskPayload {
	return taskPayload{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, Summary: t.Summary, Status: t.Status,
		ProviderID: t.ProviderID, Model: t.Model, Effort: string(t.Effort), Source: t.Source, Team: t.Team,
		PRD: t.PRD, Proof: t.Proof,
		Plan:      append([]domain.PlanStep{}, t.Plan...),
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, ContextUsed: t.ContextUsed, ContextWindow: t.ContextWindow,
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
		Team       *bool  `json:"team"`
		ProjectID  string `json:"projectId"`
		Prompt     string `json:"prompt"`
		ProviderID string `json:"providerId"`
		Model      string `json:"model"`
		Effort     string `json:"effort"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	task, err := h.service.Create(c.Context(), httpserver.UserID(c), application.CreateInput{
		ProjectID: body.ProjectID, Prompt: body.Prompt, ProviderID: body.ProviderID, Model: body.Model,
		Effort: domain.Effort(body.Effort), Team: body.Team,
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
