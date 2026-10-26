package http

import (
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

type requestInput struct {
	TaskID string `json:"taskId"`
	Host   string `json:"host"`
	Reason string `json:"reason"`
}
type approveInput struct {
	TTLHours int `json:"ttlHours"`
}
type grantView struct {
	ExpiresAt       time.Time        `json:"expiresAt"`
	CreatedAt       time.Time        `json:"createdAt"`
	ID              string           `json:"id"`
	ProjectID       string           `json:"projectId"`
	TaskID          string           `json:"taskId"`
	Host            string           `json:"host"`
	Reason          string           `json:"reason"`
	Status          domain.Status    `json:"status"`
	Requester       string           `json:"requester"`
	RequestedByKind domain.ActorKind `json:"requestedByKind"`
	DecidedBy       string           `json:"decidedBy"`
}

func view(g domain.Grant) grantView {
	return grantView{
		ID: g.ID, ProjectID: g.ProjectID, TaskID: g.TaskID, Host: g.Host, Reason: g.Reason,
		Status: g.Effective(time.Now()), Requester: g.RequestedBy.ID, RequestedByKind: g.RequestedBy.Kind,
		DecidedBy: g.DecidedBy, ExpiresAt: g.ExpiresAt, CreatedAt: g.CreatedAt,
	}
}

func (h *Handler) csrfToken(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"token": h.csrf.Token(c)})
}

func (h *Handler) list(c fiber.Ctx) error {
	grants, err := h.service.ListProject(c.Context(), c.Params("id"))
	if err != nil {
		return problem(c, err)
	}
	out := make([]grantView, 0, len(grants))
	for _, g := range grants {
		out = append(out, view(g))
	}
	return c.JSON(out)
}

func (h *Handler) get(c fiber.Ctx) error {
	grant, err := h.scoped(c)
	if err != nil {
		return problem(c, err)
	}
	return c.JSON(view(grant))
}

func (h *Handler) request(c fiber.Ctx) error {
	var body requestInput
	if err := c.Bind().JSON(&body); err != nil {
		return errorJSON(c, fiber.StatusBadRequest, "invalid_body")
	}
	grant, _, err := h.service.Ask(c.Context(), c.Params("id"), body.TaskID, body.Host, body.Reason,
		domain.Actor{Kind: domain.ActorPerson, ID: httpserver.UserID(c)})
	if err != nil {
		return problem(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(view(grant))
}

func (h *Handler) decide(c fiber.Ctx, kind string) error {
	if _, err := h.scoped(c); err != nil {
		return problem(c, err)
	}
	var (
		grant domain.Grant
		err   error
	)
	by := h.decider(c)
	switch kind {
	case "approve":
		var body approveInput
		if len(c.Body()) != 0 {
			if err = c.Bind().JSON(&body); err != nil {
				return errorJSON(c, fiber.StatusBadRequest, "invalid_body")
			}
		}
		if body.TTLHours < 0 || body.TTLHours > 168 {
			return errorJSON(c, fiber.StatusBadRequest, "invalid_ttl")
		}
		grant, err = h.service.Approve(c.Context(), c.Params("grant"), by, time.Duration(body.TTLHours)*time.Hour)
	case "deny":
		grant, err = h.service.Deny(c.Context(), c.Params("grant"), by)
	case "revoke":
		grant, err = h.service.Revoke(c.Context(), c.Params("grant"), by)
	}
	if err != nil {
		return problem(c, err)
	}
	return c.JSON(view(grant))
}
func (h *Handler) approve(c fiber.Ctx) error { return h.decide(c, "approve") }
func (h *Handler) deny(c fiber.Ctx) error    { return h.decide(c, "deny") }
func (h *Handler) revoke(c fiber.Ctx) error  { return h.decide(c, "revoke") }
