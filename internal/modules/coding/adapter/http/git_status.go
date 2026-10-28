package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
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

// gitHead returns one bounded original text file from the authorized project's HEAD.
func (h *Handler) gitHead(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}
	if workspace.Head == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(httpserver.Failure("git_unavailable"))
	}
	baseline, err := workspace.Head.Read(c.Query("path"))
	if err != nil {
		if errors.Is(err, domain.ErrDenied) || errors.Is(err, domain.ErrIsDir) {
			return fail(c, err)
		}
		return c.Status(fiber.StatusServiceUnavailable).JSON(httpserver.Failure("git_unavailable"))
	}
	return c.JSON(baseline)
}
