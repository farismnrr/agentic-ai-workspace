package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/sqlite"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

const (
	verifier    = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge   = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	resourceURI = "https://example.test/api/coding"
	redirectURI = "https://chatgpt.com/connector_platform_oauth_redirect"
	userID      = "user-1"
)

type catalog struct{}

func (catalog) Find(uri string) (application.Resource, bool) {
	return application.Resource{URI: resourceURI, Scopes: []domain.Scope{{ID: "coding:read", Description: "Read"}}},
		uri == resourceURI
}

func (c catalog) All() []application.Resource {
	resource, _ := c.Find(resourceURI)

	return []application.Resource{resource}
}

type users struct{}

func (users) Profile(context.Context, string) (application.Profile, error) {
	return application.Profile{ID: userID, Name: "Sari", Email: "sari@example.com"}, nil
}

type signer struct{}

func (signer) Sign(application.IDClaims) (string, error) { return "id.token", nil }

type noFetch struct{}

func (noFetch) Fetch(context.Context, string) (domain.Client, error) {
	return domain.Client{}, domain.ErrClientNotFound
}

func setup(t *testing.T) (*application.Service, domain.Client) {
	t.Helper()

	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })
	seedUser(t, db)

	store := sqlite.NewStore(db)
	service := application.NewService(application.Deps{
		Issuer: "https://example.test", Clients: store, Requests: store, Grants: store, Credentials: store,
		Metadata: noFetch{}, Signer: signer{}, Users: users{}, Resources: catalog{}, Tokens: opaque.Generator{},
		Now: time.Now,
	})

	client, err := service.RegisterClient(context.Background(),
		application.Registration{Name: "ChatGPT", RedirectURIs: []string{redirectURI}})
	if err != nil {
		t.Fatal(err)
	}

	return service, client
}

func seedUser(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO users (id, name, email, password_hash, created_at) VALUES (?, 'Sari', 's@x.io', 'x', 0)`, userID)
	if err != nil {
		t.Fatal(err)
	}
}

func approve(t *testing.T, service *application.Service, client domain.Client) string {
	t.Helper()

	ctx := context.Background()

	requestID, err := service.Authorize(ctx, application.AuthorizeParams{
		ResponseType: "code", ClientID: client.ID, RedirectURI: redirectURI, Scope: "openid email coding:read",
		State: "xyz", CodeChallenge: challenge, CodeChallengeMethod: "S256", Resource: resourceURI,
	})
	if err != nil {
		t.Fatal(err)
	}

	location, err := service.Decide(ctx, userID, requestID, true)
	if err != nil {
		t.Fatal(err)
	}

	parsed, _ := url.Parse(location)
	if parsed.Query().Get("state") != "xyz" || parsed.Query().Get("iss") != "https://example.test" {
		t.Fatalf("redirect must carry state and iss: %s", location)
	}

	return parsed.Query().Get("code")
}

func exchange(service *application.Service, client domain.Client, code string) (application.TokenResponse, error) {
	tokens, err := service.ExchangeCode(context.Background(), application.ExchangeParams{
		Code: code, RedirectURI: redirectURI, ClientID: client.ID, CodeVerifier: verifier, Resource: resourceURI,
	})
	if err != nil {
		return application.TokenResponse{}, fmt.Errorf("exchange: %w", err)
	}

	return tokens, nil
}
