package http

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Authenticator resolves the browser session cookie to a user id. It is the
// consumer-owned boundary to the account feature.
type Authenticator interface {
	UserID(ctx context.Context, sessionToken string) (string, error)
}

// KeySet publishes the public keys that verify ID tokens.
type KeySet interface {
	JWKS() map[string]any
}

// Handler serves the OAuth 2.1 authorization server endpoints.
type Handler struct {
	service  *application.Service
	auth     Authenticator
	keys     KeySet
	metadata serverMetadata
}

// NewHandler builds the OAuth HTTP adapter for the given issuer.
func NewHandler(issuer string, service *application.Service, auth Authenticator, keys KeySet,
	resources []application.Resource,
) *Handler {
	return &Handler{service: service, auth: auth, keys: keys, metadata: newMetadata(issuer, resources)}
}

// Mount registers discovery, protocol and consent routes. Protocol endpoints
// carry no cookies, so cors (not sameOrigin) guards them for browser-based MCP clients.
func (h *Handler) Mount(sameOrigin, cors fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Use("/.well-known", cors)
		app.Use("/oauth", cors)

		app.Get("/.well-known/oauth-authorization-server", h.serveMetadata)
		app.Get("/.well-known/openid-configuration", h.serveMetadata)
		app.Get("/oauth/jwks.json", h.serveKeys)
		app.Get("/oauth/authorize", h.authorize)
		app.Post("/oauth/token", h.token)
		app.Post("/oauth/register", h.register)
		app.Post("/oauth/revoke", h.revoke)
		app.All("/oauth/userinfo", h.userinfo)

		api := app.Group("/api/oauth", sameOrigin)
		api.Get("/requests/:id", h.describe)
		api.Post("/requests/:id/decision", h.decide)
		api.Get("/grants", h.grants)
		api.Delete("/grants/:id", h.revokeGrant)
	}
}

type errorBody struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

// writeError renders an OAuth error as JSON; unknown errors become server_error.
func writeError(c fiber.Ctx, err error) error {
	oauthErr, ok := domain.AsError(err)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(errorBody{Error: domain.CodeServerError})
	}

	status := fiber.StatusBadRequest
	if oauthErr.Code == domain.CodeServerError {
		status = fiber.StatusInternalServerError
	}

	return c.Status(status).JSON(errorBody{Error: oauthErr.Code, Description: oauthErr.Description})
}

func noStore(c fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderPragma, "no-cache")
}

var errNoBearer = errors.New("missing bearer token")

func bearer(c fiber.Ctx) (string, error) {
	const prefix = "Bearer "

	header := c.Get(fiber.HeaderAuthorization)
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", errNoBearer
	}

	return header[len(prefix):], nil
}
