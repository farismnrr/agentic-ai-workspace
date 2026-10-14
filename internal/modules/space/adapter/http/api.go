package http

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Sessions resolves a browser session cookie to a user id.
type Sessions interface {
	UserID(ctx context.Context, sessionToken string) (string, error)
}

// API serves the app's own endpoints for spaces: listing, tool switches and
// the activity log. It is separate from the MCP endpoints AI apps use.
type API struct {
	registry    *application.Registry
	sessions    Sessions
	permissions *application.Permissions
	activity    *application.ActivityLog
}

// NewAPI builds the app-facing space API.
func NewAPI(
	registry *application.Registry, sessions Sessions, permissions *application.Permissions,
	activity *application.ActivityLog,
) *API {
	return &API{registry: registry, sessions: sessions, permissions: permissions, activity: activity}
}

// Mount registers the routes; sameOrigin guards the state-changing ones.
func (a *API) Mount(sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/spaces", a.list)
		app.Put("/api/spaces/:slug/tools/:tool", sameOrigin, a.setTool)
		app.Get("/api/activity", a.entries)
		app.Post("/api/activity/:id/undo", sameOrigin, a.undo)
	}
}

// signedIn returns the session's user id, or writes a 401 and reports false.
func (a *API) signedIn(c fiber.Ctx) (string, bool) {
	userID, err := a.sessions.UserID(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("unauthenticated"))

		return "", false
	}

	return userID, true
}

func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrUnknownSpace), errors.Is(err, domain.ErrUnknownTool),
		errors.Is(err, domain.ErrNoActivity):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	case errors.Is(err, domain.ErrAlreadyUndone):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("already_undone"))
	case errors.Is(err, domain.ErrNotUndoable):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("not_undoable"))
	case errors.Is(err, domain.ErrUndoRefused):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("undo_refused"))
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
