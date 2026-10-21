package http

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type connectionResponse struct {
	domain.Metadata
	SharedPostgres bool `json:"sharedPostgres"`
}

func (h *Handler) get(c fiber.Ctx) error {
	item, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	configured, err := h.service.SharedConfigured(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	return c.JSON(connectionResponse{Metadata: item, SharedPostgres: configured})
}
