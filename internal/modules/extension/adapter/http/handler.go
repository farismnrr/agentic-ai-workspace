// Package http serves the extensions API: the list of what the AI built and the package the person downloads.
package http

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Service is what the adapter needs from the extension use cases.
type Service interface {
	List(ctx context.Context, userID string) ([]domain.Extension, error)
	Package(ctx context.Context, userID, projectID string) (domain.Extension, io.ReadCloser, int64, error)
	Remove(ctx context.Context, userID, projectID string) error
}

// Handler serves /api/extensions.
type Handler struct{ service Service }

// NewHandler builds the adapter.
func NewHandler(service Service) *Handler { return &Handler{service: service} }

// Mount registers the routes.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		api := app.Group("/api/extensions", auth)
		api.Get("/", h.list)
		api.Get("/:project/package", h.download)
		api.Delete("/:project", sameOrigin, h.remove)
	}
}

type payload struct {
	DeployedAt   time.Time           `json:"deployedAt"`
	ProjectID    string              `json:"projectId"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Description  string              `json:"description"`
	FileName     string              `json:"fileName"`
	DownloadURL  string              `json:"downloadUrl"`
	Capabilities []domain.Capability `json:"capabilities"`
	Size         int64               `json:"size"`
	Files        int                 `json:"files"`
}

func toPayload(e domain.Extension) payload {
	manifest := domain.Manifest{Permissions: e.Permissions, HostPermissions: e.Hosts}

	return payload{
		ProjectID: e.ProjectID, Name: e.Name, Version: e.Version, Description: e.Description, FileName: e.FileName(),
		DownloadURL: "/api/extensions/" + url.PathEscape(e.ProjectID) + "/package", Capabilities: manifest.Capabilities(),
		Size: e.Size, Files: e.Files, DeployedAt: e.DeployedAt,
	}
}

func (h *Handler) list(c fiber.Ctx) error {
	extensions, err := h.service.List(c.Context(), httpserver.UserID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	out := make([]payload, 0, len(extensions))
	for _, extension := range extensions {
		out = append(out, toPayload(extension))
	}

	return c.JSON(fiber.Map{"extensions": out})
}

func (h *Handler) download(c fiber.Ctx) error {
	extension, file, size, err := h.service.Package(c.Context(), httpserver.UserID(c), c.Params("project"))
	if err != nil {
		return fail(c, err)
	}

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", `attachment; filename="`+extension.FileName()+`"`)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cache-Control", "no-store")

	return c.SendStream(file, int(size))
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Remove(c.Context(), httpserver.UserID(c), c.Params("project")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c fiber.Ctx, err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
