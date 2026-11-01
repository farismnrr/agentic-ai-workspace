package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// enableExternalDeepReview is an authenticated, CSRF-protected user action,
// not a connected tool. The application checks task ownership and source.
func (h *Handler) enableExternalDeepReview(c fiber.Ctx) error {
	task, err := h.service.EnableExternalDeepReview(
		c.Context(), httpserver.UserID(c), c.Params("id"),
	)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(toPayload(task))
}
