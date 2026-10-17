package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// create makes a new repository on the person's GitHub account.
func (h *Handler) create(c fiber.Ctx) error {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_name"))
	}

	repo, err := h.service.CreateRepo(c.Context(), httpserver.UserID(c),
		domain.NewRepo{Name: in.Name, Description: in.Description, Private: in.Private})

	switch {
	case err == nil:
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"name": repo.FullName, "private": repo.Private})
	case errors.Is(err, domain.ErrBadRepoName):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_name"))
	case errors.Is(err, domain.ErrRepoExists):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("exists"))
	case errors.Is(err, domain.ErrNotConnected), errors.Is(err, domain.ErrRejected):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("not_connected"))
	case errors.Is(err, domain.ErrCannotCreate):
		return c.Status(fiber.StatusForbidden).JSON(httpserver.Failure("cannot_create"))
	}

	return c.Status(fiber.StatusBadGateway).JSON(httpserver.Failure("github_unavailable"))
}
