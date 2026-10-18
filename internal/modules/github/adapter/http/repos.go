package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type repoPayload struct {
	UpdatedAt   time.Time `json:"updatedAt"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
}

// repos lists what the person let the app reach, so a project can be made from one.
func (h *Handler) repos(c fiber.Ctx) error {
	repos, err := h.service.Repos(c.Context(), httpserver.UserID(c))
	if errors.Is(err, domain.ErrNotConnected) || errors.Is(err, domain.ErrRejected) {
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("not_connected"))
	}

	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(httpserver.Failure("github_unavailable"))
	}

	out := make([]repoPayload, 0, len(repos))
	for _, repo := range repos {
		out = append(out, repoPayload{
			Name: repo.FullName, Description: repo.Description, Private: repo.Private, UpdatedAt: repo.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{"repos": out})
}
