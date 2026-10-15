package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// ClientKey names the token field that carries the connected app's name.
const ClientKey = "client"

// Access describes a verified bearer token.
type Access struct {
	ExpiresAt time.Time
	UserID    string
	Resource  string
	// Client is the name of the app the person connected, such as ChatGPT.
	Client string
	Scopes []string
}

// TokenVerifier is the consumer-owned boundary to the authorization server.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Access, error)
}

// Mounter publishes spaces over HTTP.
type Mounter struct {
	verifier  TokenVerifier
	registry  *application.Registry
	publicURL string
	issuer    string
}

// NewMounter builds the space HTTP adapter. issuer is the authorization server advertised in metadata.
func NewMounter(registry *application.Registry, publicURL, issuer string, verifier TokenVerifier) *Mounter {
	return &Mounter{registry: registry, publicURL: publicURL, issuer: issuer, verifier: verifier}
}

// Mount serves one space: its Protected Resource Metadata (open) and its MCP
// endpoint behind bearer-token authentication bound to this space's resource.
func (m *Mounter) Mount(space domain.Space, mcpHandler http.Handler) httpserver.Mount {
	resource := space.ResourceURI(m.publicURL)
	metadataURL := m.publicURL + space.MetadataPath()

	metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               resource,
		AuthorizationServers:   []string{m.issuer},
		ScopesSupported:        space.ScopeIDs(),
		BearerMethodsSupported: []string{"header"},
		ResourceName:           space.Name,
	})

	protected := auth.RequireBearerToken(m.verify(resource), &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: metadataURL,
		Scopes:              []string{space.Entry},
	})(mcpHandler)
	protected = withScopeError(protected)

	return func(app *fiber.App) {
		app.Get(space.MetadataPath(), adaptor.HTTPHandler(metadata))
		app.All(space.Path(), adaptor.HTTPHandler(protected))
	}
}

var errAudience = errors.New("token was issued for a different resource")

// verify accepts only tokens minted for this space's resource (audience validation).
func (m *Mounter) verify(resource string) auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		access, err := m.verifier.Verify(ctx, token)
		if err != nil {
			return nil, errors.Join(auth.ErrInvalidToken, err)
		}

		if access.Resource != resource {
			return nil, errors.Join(auth.ErrInvalidToken, errAudience)
		}

		return &auth.TokenInfo{
			Scopes: access.Scopes, Expiration: access.ExpiresAt, UserID: access.UserID,
			Extra: map[string]any{ClientKey: access.Client},
		}, nil
	}
}
