package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// setContext records how much one of a provider's models can hold at once. The model
// travels in the body because names such as "cx/gpt-6[1m]" do not fit a path.
func (h *Handler) setContext(c fiber.Ctx) error {
	var body struct {
		Model         string `json:"model"`
		ContextWindow int64  `json:"contextWindow"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	err := h.service.SetContextWindow(c.Context(), httpserver.UserID(c), c.Params("id"), body.Model, body.ContextWindow)
	if err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
