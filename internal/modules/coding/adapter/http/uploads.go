package http

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type uploadResponse struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Media string `json:"media"`
	Size  int64  `json:"size"`
}

// uploadFile accepts one validated project file from the multipart field "file".
func (h *Handler) uploadFile(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}
	header, err := c.FormFile("file")
	if err != nil {
		return fail(c, domain.ErrEmpty)
	}
	file, err := header.Open()
	if err != nil {
		return fail(c, fmt.Errorf("open multipart file: %w", err))
	}
	defer func() { _ = file.Close() }()

	stored, err := workspace.Editor.Upload(header.Filename, header.Size, file)
	if err != nil {
		return fail(c, err)
	}
	h.changes.Announce(c.Params("id"))
	return c.Status(fiber.StatusCreated).JSON(uploadResponse{
		Path: stored.Path, Name: stored.Name, Kind: stored.Kind, Media: stored.Media, Size: stored.Size,
	})
}
