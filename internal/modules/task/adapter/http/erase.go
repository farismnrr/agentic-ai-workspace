package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func (h *Handler) deleteAll(c fiber.Ctx) error {
	if err := h.service.DeleteAll(c.Context(), httpserver.UserID(c)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}
