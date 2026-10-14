// Package http serves the data export.
package http

import (
	"bufio"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/adapter/zip"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves GET /api/account/export.
type Handler struct {
	exporter *application.Exporter
	now      func() time.Time
}

// NewHandler builds the export adapter.
func NewHandler(exporter *application.Exporter, now func() time.Time) *Handler {
	return &Handler{exporter: exporter, now: now}
}

// Mount registers the route behind the session guard.
func (h *Handler) Mount(auth fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) { app.Get("/api/account/export", auth, h.export) }
}

func (h *Handler) export(c fiber.Ctx) error {
	userID, now := httpserver.UserID(c), h.now()

	c.Set(fiber.HeaderContentType, "application/zip")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="my-data-`+now.Format("2006-01-02")+`.zip"`)
	c.Set(fiber.HeaderCacheControl, "no-store")

	ctx := c.Context()

	return c.SendStreamWriter(func(w *bufio.Writer) {
		archive := zip.New(w, now)
		if err := h.exporter.Export(ctx, userID, archive); err != nil {
			slog.Error("data export failed", "error", err)
		}

		if err := archive.Close(); err != nil {
			slog.Error("data export failed", "error", err)
		}
	})
}
