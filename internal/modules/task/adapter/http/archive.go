package http

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type batchRequest struct {
	IDs []string `json:"ids"`
}

const maxBatchIDs = 1000

func (h *Handler) archived(c fiber.Ctx) error {
	tasks, err := h.service.Archived(c.Context(), httpserver.UserID(c))
	return sendList(c, tasks, err, toPayload)
}

func (h *Handler) archive(c fiber.Ctx) error {
	return h.batch(c, h.service.ArchiveTasks)
}

func (h *Handler) restore(c fiber.Ctx) error {
	return h.batch(c, h.service.RestoreTasks)
}

func (h *Handler) deleteTasks(c fiber.Ctx) error {
	return h.batch(c, h.service.DeleteTasks)
}

type batchAction func(ctx context.Context, userID string, ids []string) []application.BatchResult

func (h *Handler) batch(c fiber.Ctx, action batchAction) error {
	var body batchRequest
	if err := c.Bind().JSON(&body); err != nil || body.IDs == nil {
		return badBody(c)
	}
	if len(body.IDs) > maxBatchIDs {
		return badBody(c)
	}
	return c.JSON(action(c.Context(), httpserver.UserID(c), uniqueIDs(body.IDs)))
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
