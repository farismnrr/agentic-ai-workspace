package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type planDecisionBody struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Note           string `json:"note"`
	Revision       int64  `json:"revision"`
}

type planDecisionPayload struct {
	Decision     string    `json:"decision"`
	Source       string    `json:"source,omitempty"`
	DecidedAt    time.Time `json:"decidedAt,omitempty"`
	HandoffID    string    `json:"handoffId,omitempty"`
	HandoffState string    `json:"handoffState,omitempty"`
	LinkedTaskID string    `json:"linkedTaskId,omitempty"`
}

const decisionKeyBytes = 16

func (h *Handler) approvePlan(c fiber.Ctx) error {
	var body planDecisionBody
	if err := c.Bind().JSON(&body); err != nil || body.Revision < 1 {
		return badBody(c)
	}
	key, err := decisionKey(body.IdempotencyKey)
	if err != nil {
		return fail(c, err)
	}
	input, err := h.service.PlanReviewInput(c.Context(), httpserver.UserID(c), c.Params("id"),
		body.Revision, key, "")
	if err != nil {
		return fail(c, err)
	}
	result, err := h.service.ApprovePlanFromApp(c.Context(), httpserver.UserID(c), input)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(planDecisionPayload{
		Decision: string(result.Decision.Kind), Source: result.Decision.Source, DecidedAt: result.Decision.DecidedAt,
		HandoffID:    result.HandoffID,
		HandoffState: result.HandoffState, LinkedTaskID: result.InternalTaskID,
	})
}

func (h *Handler) requestPlanChanges(c fiber.Ctx) error {
	var body planDecisionBody
	if err := c.Bind().JSON(&body); err != nil || body.Revision < 1 {
		return badBody(c)
	}
	key, err := decisionKey(body.IdempotencyKey)
	if err != nil {
		return fail(c, err)
	}
	input, err := h.service.PlanReviewInput(c.Context(), httpserver.UserID(c), c.Params("id"),
		body.Revision, key, body.Note)
	if err != nil {
		return fail(c, err)
	}
	result, err := h.service.RequestPlanChanges(c.Context(), httpserver.UserID(c), input)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(planDecisionPayload{
		Decision: string(result.Decision.Kind), Source: result.Decision.Source, DecidedAt: result.Decision.DecidedAt,
	})
}

func (h *Handler) planRevision(c fiber.Ctx) error {
	revision, err := strconv.ParseInt(c.Params("rev"), 10, 64)
	if err != nil || revision < 1 {
		return badBody(c)
	}
	snapshot, err := h.service.PlanReviewContext(c.Context(), httpserver.UserID(c), c.Params("id"), revision)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(snapshot)
}

func decisionKey(key string) (string, error) {
	if key != "" {
		return key, nil
	}
	bytes := make([]byte, decisionKeyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate plan decision key: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
