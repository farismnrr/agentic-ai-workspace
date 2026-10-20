package http

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

func (h *Handler) documents(c fiber.Ctx) error {
	docs, err := h.service.Documents(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(docs)
}
