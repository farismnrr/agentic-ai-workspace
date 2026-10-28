package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// gitStatus returns the current, bounded Git snapshot for the authorized project.
func (h *Handler) gitStatus(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	if workspace.Git == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(httpserver.Failure("git_unavailable"))
	}
	snapshot, err := workspace.Git.Read()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(httpserver.Failure("git_unavailable"))
	}

	return c.JSON(snapshot)
}
