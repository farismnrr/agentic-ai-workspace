package http

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const (
	uploadBodyAllowance  = 1 << 20
	uploadReadTimeout    = 10 * time.Minute
	maxConcurrentUploads = 4
)

// Handler serves the file endpoints of a project.
type Handler struct {
	workspaces Workspaces
	changes    Changes
}

// NewHandler builds the file HTTP adapter.
func NewHandler(workspaces Workspaces, changes Changes) *Handler {
	return &Handler{workspaces: workspaces, changes: changes}
}

// Mount registers the routes; auth guards all, sameOrigin the ones that change a file.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler, session httpserver.Authenticator) httpserver.Mount {
	return func(app *fiber.App) {
		httpserver.RaiseBodyLimit(app, httpserver.BodyRule{
			Method: "POST", Path: "/uploads", ContentType: "multipart/form-data",
			MaxBodySize: int(domain.MaxVideoBytes + uploadBodyAllowance), ReadTimeout: uploadReadTimeout,
		}, session)
		app.Get("/api/projects/:id/files", auth, h.list)
		app.Get("/api/projects/:id/files/stream", auth, h.stream)
		app.Get("/api/projects/:id/file", auth, h.read)
		app.Get("/api/projects/:id/file/raw", auth, h.raw)
		app.Post("/api/projects/:id/assets", auth, sameOrigin, h.upload)
		app.Post("/api/projects/:id/uploads", auth, sameOrigin, uploadConcurrency(maxConcurrentUploads), h.uploadFile)
		app.Put("/api/projects/:id/file", auth, sameOrigin, h.save)
		app.Delete("/api/projects/:id/file", auth, sameOrigin, h.remove)
	}
}

var mapped = []struct {
	err    error
	code   string
	status int
}{
	{domain.ErrNotFound, "not_found", fiber.StatusNotFound},
	{domain.ErrDenied, "not_found", fiber.StatusNotFound},
	{domain.ErrIsDir, "bad_request", fiber.StatusBadRequest},
	{domain.ErrNotDir, "bad_request", fiber.StatusBadRequest},
	{domain.ErrTooLarge, "too_large", fiber.StatusRequestEntityTooLarge},
	{domain.ErrUnsupported, "unsupported_type", fiber.StatusUnsupportedMediaType},
	{domain.ErrEmpty, "empty", fiber.StatusBadRequest},
	{domain.ErrStale, "changed", fiber.StatusConflict},
	{domain.ErrExists, "exists", fiber.StatusConflict},
	{domain.ErrBinary, "binary", fiber.StatusUnsupportedMediaType},
}

func fail(c fiber.Ctx, err error) error {
	for _, known := range mapped {
		if errors.Is(err, known.err) {
			return c.Status(known.status).JSON(httpserver.Failure(known.code))
		}
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("server_error"))
}

// open finds the person's project, or answers not found for a project that is not theirs.
func (h *Handler) open(c fiber.Ctx) (Workspace, error) {
	workspace, err := h.workspaces.Open(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return Workspace{}, fmt.Errorf("open the project: %w", err)
	}

	return workspace, nil
}
