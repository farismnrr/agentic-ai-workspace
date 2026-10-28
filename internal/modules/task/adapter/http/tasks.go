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
	PinnedAt   *time.Time    `json:"pinnedAt"`
	ArchivedAt *time.Time    `json:"archivedAt"`
	Proof      *domain.Proof `json:"proof,omitempty"`
	ProviderID string        `json:"providerId"`
	ProjectID  string        `json:"projectId"`
	Title      string        `json:"title"`
	Summary    string        `json:"summary"`
	// PRD is the requirements document the team's Planner wrote, in markdown; empty when there is none.
	PRD string `json:"prd"`
	// Research is what the team found out from the web and the project, in markdown; empty when none.
	Research     string              `json:"research"`
	Status       domain.Status       `json:"status"`
	ID           string              `json:"id"`
	Model        string              `json:"model"`
	Effort       string              `json:"effort"`
	ReviewMode   domain.ReviewMode   `json:"mode"`
	Source       string              `json:"source"`
	WorkID       string              `json:"workId,omitempty"`
	LinkedTaskID string              `json:"linkedTaskId,omitempty"`
	OriginTaskID string              `json:"originTaskId,omitempty"`
	Work         *domain.WorkSummary `json:"work,omitempty"`
	Plan         []domain.PlanStep   `json:"plan"`
	// ContextUsed and ContextWindow are tokens; both are zero before the first model call.
	ContextUsed   int64 `json:"contextUsed"`
	ContextWindow int64 `json:"contextWindow"`
	// Team is true when the agent leads a team of teammates on this task.
	Team bool `json:"team"`
	// PlanFirst is true when the team stops at the PRD and waits for the person.
	PlanFirst bool `json:"planFirst"`
}

func toPayload(t domain.Task) taskPayload {
	return taskPayload{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, Summary: t.Summary, Status: t.Status,
		ProviderID: t.ProviderID, Model: t.Model, Effort: string(t.Effort), ReviewMode: t.ReviewMode,
		Source: t.Source, Team: t.Team,
		WorkID: t.WorkID, Work: t.Work, LinkedTaskID: t.LinkedTaskID, OriginTaskID: t.OriginTaskID,
		PlanFirst: t.PlanFirst,
		PRD:       t.PRD, Research: t.Research, Proof: t.Proof,
		Plan:      append([]domain.PlanStep{}, t.Plan...),
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, PinnedAt: t.PinnedAt, ArchivedAt: t.ArchivedAt,
		ContextUsed: t.ContextUsed, ContextWindow: t.ContextWindow,
	}
}

// sendList answers with a list converted to its payload shape, or with the failure that stopped it.
func sendList[T, P any](c fiber.Ctx, items []T, err error, convert func(T) P) error {
	if err != nil {
		return fail(c, err)
	}

	out := make([]P, 0, len(items))
	for _, item := range items {
		out = append(out, convert(item))
	}

	return c.JSON(out)
}

func (h *Handler) list(c fiber.Ctx) error {
	tasks, err := h.service.List(c.Context(), httpserver.UserID(c), c.Query("project"))

	return sendList(c, tasks, err, toPayload)
}

func (h *Handler) get(c fiber.Ctx) error {
	task, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(task))
}

func (h *Handler) pin(c fiber.Ctx) error {
	task, err := h.service.Pin(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(task))
}

func (h *Handler) unpin(c fiber.Ctx) error {
	task, err := h.service.Unpin(c.Context(), httpserver.UserID(c), c.Params("id"))
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
		ReviewMode string `json:"mode"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	task, err := h.service.Create(c.Context(), httpserver.UserID(c), application.CreateInput{
		ProjectID: body.ProjectID, Prompt: body.Prompt, ProviderID: body.ProviderID, Model: body.Model,
		Effort: domain.Effort(body.Effort), Team: body.Team, ReviewMode: body.ReviewMode,
	})
	if err != nil {
		return fail(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toPayload(task))
}

func (h *Handler) message(c fiber.Ctx) error {
	var body struct {
		Text       string `json:"text"`
		ReviewMode string `json:"mode"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}

	return h.done(c, h.service.FollowUp(c.Context(), httpserver.UserID(c), c.Params("id"),
		application.FollowUpInput{Text: body.Text, ReviewMode: body.ReviewMode}))
}

func (h *Handler) externalChat(c fiber.Ctx) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return badBody(c)
	}
	linkedID, err := h.service.ChatOnExternal(c.Context(), httpserver.UserID(c), c.Params("id"), body.Text)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(struct {
		LinkedTaskID string `json:"linkedTaskId"`
	}{LinkedTaskID: linkedID})
}

func (h *Handler) cancel(c fiber.Ctx) error {
	return h.done(c, h.service.Cancel(c.Context(), httpserver.UserID(c), c.Params("id")))
}

func (h *Handler) confirm(c fiber.Ctx) error {
	result, err := h.service.Confirm(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(result)
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
