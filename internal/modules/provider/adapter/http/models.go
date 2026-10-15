package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// models lists every model a person can pick, from all their enabled and verified providers.
func (h *Handler) models(c fiber.Ctx) error {
	h.service.SyncStale(c.Context(), httpserver.UserID(c))

	options, err := h.service.Models(c.Context(), httpserver.UserID(c))
	if err != nil {
		return fail(c, err)
	}

	type option struct {
		ProviderID   string `json:"providerId"`
		ProviderName string `json:"providerName"`
		Model        string `json:"model"`
		Group        string `json:"group"`
	}

	payload := make([]option, 0, len(options))
	for _, o := range options {
		payload = append(payload, option{
			ProviderID: o.ProviderID, ProviderName: o.ProviderName, Model: o.Model, Group: o.Group,
		})
	}

	return c.JSON(payload)
}
