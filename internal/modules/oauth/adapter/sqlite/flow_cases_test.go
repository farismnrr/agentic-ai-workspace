package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

func issue(t *testing.T, service *application.Service, client domain.Client) application.TokenResponse {
	t.Helper()

	tokens, err := exchange(service, client, approve(t, service, client))
	if err != nil {
		t.Fatal(err)
	}

	return tokens
}

func TestOpenIDGrantReturnsIDTokenAndRefreshToken(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	tokens := issue(t, service, client)

	if tokens.IDToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("openid grant must return an id_token and refresh token: %+v", tokens)
	}
}

func TestAccessTokenVerifiesForItsResource(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	tokens := issue(t, service, client)

	info, err := service.VerifyAccessToken(context.Background(), tokens.AccessToken)
	if err != nil || info.Resource != resourceURI || info.UserID != userID {
		t.Fatalf("access token should verify for the resource: %+v %v", info, err)
	}
}

// Replaying a rotated-out refresh token is treated as theft: the grant dies.
func TestRefreshReuseRevokesGrant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, client := setup(t)
	tokens := issue(t, service, client)
	params := application.RefreshParams{RefreshToken: tokens.RefreshToken, ClientID: client.ID}

	rotated, err := service.Refresh(ctx, params)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Refresh(ctx, params)
	if oauthErr, ok := domain.AsError(err); !ok || oauthErr.Code != domain.CodeInvalidGrant {
		t.Fatalf("refresh reuse must be invalid_grant, got %v", err)
	}

	if _, err := service.VerifyAccessToken(ctx, rotated.AccessToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("grant should be revoked after refresh reuse, got %v", err)
	}
}

func TestCodeReplayRevokesGrant(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	code := approve(t, service, client)

	tokens, err := exchange(service, client, code)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := exchange(service, client, code); err == nil {
		t.Fatal("replayed code must be rejected")
	}

	if _, err := service.VerifyAccessToken(context.Background(), tokens.AccessToken); err == nil {
		t.Fatal("tokens from a replayed code must be revoked")
	}
}

func TestExchangeRejectsBadBindings(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	ctx := context.Background()

	cases := map[string]application.ExchangeParams{
		"bad verifier":   {RedirectURI: redirectURI, ClientID: client.ID, CodeVerifier: verifier[:42] + "x"},
		"redirect":       {RedirectURI: redirectURI + "2", ClientID: client.ID, CodeVerifier: verifier},
		"other client":   {RedirectURI: redirectURI, ClientID: "other", CodeVerifier: verifier},
		"wrong resource": {RedirectURI: redirectURI, ClientID: client.ID, CodeVerifier: verifier, Resource: "https://x/y"},
	}

	for name, params := range cases {
		params.Code = approve(t, service, client)

		if _, err := service.ExchangeCode(ctx, params); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestAuthorizeValidation(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	base := application.AuthorizeParams{
		ResponseType: "code", ClientID: client.ID, RedirectURI: redirectURI, Scope: "coding:read",
		CodeChallenge: challenge, CodeChallengeMethod: "S256", Resource: resourceURI,
	}

	bad := base
	bad.RedirectURI = "https://evil.test/cb"

	_, err := service.Authorize(context.Background(), bad)
	if oauthErr, ok := domain.AsError(err); !ok || oauthErr.RedirectURI != "" {
		t.Fatalf("unregistered redirect_uri must not be redirectable: %v", err)
	}

	for name, mutate := range map[string]func(*application.AuthorizeParams){
		"plain pkce": func(p *application.AuthorizeParams) { p.CodeChallengeMethod = "plain" },
		"resource":   func(p *application.AuthorizeParams) { p.Resource = "https://other.test/mcp" },
		"scope":      func(p *application.AuthorizeParams) { p.Scope = "coding:write" },
	} {
		params := base
		mutate(&params)

		_, err := service.Authorize(context.Background(), params)
		if oauthErr, ok := domain.AsError(err); !ok || oauthErr.RedirectURI == "" {
			t.Errorf("%s must fail with a redirectable error, got %v", name, err)
		}
	}
}

func TestRevokeGrantStopsAccess(t *testing.T) {
	t.Parallel()

	service, client := setup(t)
	ctx := context.Background()

	tokens, err := exchange(service, client, approve(t, service, client))
	if err != nil {
		t.Fatal(err)
	}

	info, _ := service.VerifyAccessToken(ctx, tokens.AccessToken)
	if err := service.RevokeGrant(ctx, "someone-else", info.GrantID); !errors.Is(err, domain.ErrGrantNotFound) {
		t.Fatalf("only the owner may revoke, got %v", err)
	}

	if err := service.RevokeGrant(ctx, userID, info.GrantID); err != nil {
		t.Fatal(err)
	}

	if _, err := service.VerifyAccessToken(ctx, tokens.AccessToken); err == nil {
		t.Fatal("revoked grant must stop working immediately")
	}
}
