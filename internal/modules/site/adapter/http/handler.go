// Package http serves the sites API and the reverse proxy that opens a site's pages.
package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const publicRate = 600

// Service is what the adapter needs from the sites use cases.
type Service interface {
	List(ctx context.Context, userID string) ([]domain.Site, error)
	Stop(ctx context.Context, userID, projectID string) error
	Restart(ctx context.Context, userID, projectID string) (domain.Site, error)
	SetVisibility(ctx context.Context, userID, projectID, value string) error
	Resolve(ctx context.Context, slug string) (domain.Site, error)
	ResolveOld(ctx context.Context, token string) (domain.Site, error)
	Socket(projectID string) string
	Ticket(token, holder string) string
	ValidTicket(token, holder, ticket string) bool
}

// Handler serves /api/sites and /run.
type Handler struct {
	service Service
	auth    httpserver.Authenticator
	// bindIP also ties tickets to the visitor's address. It is only meaningful when the server can see
	// the real client address (CLIENT_IP_HEADER behind a proxy); otherwise it would vary or be shared.
	bindIP bool
}

// NewHandler builds the adapter. auth is only used to recognise the owner on /run, where a
// missing login is not an error.
func NewHandler(service Service, auth httpserver.Authenticator, bindIP bool) *Handler {
	return &Handler{service: service, auth: auth, bindIP: bindIP}
}

// Mount registers the routes. /run is outside the session and same-origin rules on purpose:
// it serves other people's code, and its safety comes from the sandbox headers instead.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		api := app.Group("/api/sites", auth)
		api.Get("/", h.list)
		api.Post("/:project/stop", sameOrigin, h.stop)
		api.Post("/:project/start", sameOrigin, h.start)
		api.Patch("/:project", sameOrigin, h.visibility)

		app.All("/run/:name/*", httpserver.Limit(publicRate, time.Minute, nil, false), h.serve)
	}
}
