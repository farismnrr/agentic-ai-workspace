package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// AuthorizeParams are the query parameters of an authorization request.
type AuthorizeParams struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	Nonce               string
}

// Authorize validates an authorization request and parks it until the person
// answers on the consent page. Errors before the redirect URI is trusted carry
// no redirect; later ones are returned to the client through it.
func (s *Service) Authorize(ctx context.Context, p AuthorizeParams) (string, error) {
	client, err := s.ResolveClient(ctx, p.ClientID)
	if err != nil {
		return "", err
	}

	redirectURI := p.RedirectURI
	if redirectURI == "" && len(client.RedirectURIs) == 1 {
		redirectURI = client.RedirectURIs[0]
	}

	if !client.AllowsRedirect(redirectURI) {
		return "", domain.NewError(domain.CodeInvalidRequest, "redirect_uri is not registered for this client")
	}

	scopes, validationErr := s.validate(p)
	if validationErr != nil {
		return "", validationErr.Redirected(redirectURI, p.State)
	}

	id, err := s.Tokens.ID()
	if err != nil {
		return "", fmt.Errorf("new request id: %w", err)
	}

	request := domain.AuthorizationRequest{
		ID: id, ClientID: client.ID, RedirectURI: redirectURI, Scopes: scopes, State: p.State,
		CodeChallenge: p.CodeChallenge, Nonce: p.Nonce, Resource: p.Resource, ExpiresAt: s.Now().Add(requestTTL),
	}
	if err := s.Requests.SaveRequest(ctx, request); err != nil {
		return "", fmt.Errorf("save request: %w", err)
	}

	return id, nil
}

func (s *Service) validate(p AuthorizeParams) ([]string, *domain.Error) {
	if p.ResponseType != "code" {
		return nil, domain.NewError(domain.CodeUnsupportedResponse, "only response_type=code is supported")
	}

	if p.CodeChallengeMethod != "S256" || !domain.ValidChallenge(p.CodeChallenge) {
		return nil, domain.NewError(domain.CodeInvalidRequest, "PKCE with code_challenge_method=S256 is required")
	}

	resource, ok := s.Resources.Find(p.Resource)
	if !ok {
		return nil, domain.NewError(domain.CodeInvalidTarget, "unknown resource")
	}

	return s.checkScopes(p.Scope, resource)
}

func (s *Service) checkScopes(raw string, resource Resource) ([]string, *domain.Error) {
	scopes := domain.SplitScopes(raw)
	if len(scopes) == 0 {
		for _, scope := range resource.Scopes {
			scopes = append(scopes, scope.ID)
		}
	}

	known := scopeDescriptions(resource)
	for _, scope := range scopes {
		if _, ok := known[scope]; !ok {
			return nil, domain.NewError(domain.CodeInvalidScope, "unknown scope "+scope)
		}
	}

	return scopes, nil
}

func scopeDescriptions(resource Resource) map[string]string {
	known := map[string]string{}

	for _, scope := range domain.IdentityScopes {
		known[scope.ID] = scope.Description
	}

	for _, scope := range resource.Scopes {
		known[scope.ID] = scope.Description
	}

	return known
}

// ConsentView is what the consent page shows.
type ConsentView struct {
	ClientName   string
	RedirectHost string
	Scopes       []domain.Scope
}

// Describe loads a pending request for the consent page.
func (s *Service) Describe(ctx context.Context, requestID string) (ConsentView, error) {
	request, err := s.pending(ctx, requestID, s.Now())
	if err != nil {
		return ConsentView{}, err
	}

	client, err := s.Clients.FindClient(ctx, request.ClientID)
	if err != nil {
		return ConsentView{}, fmt.Errorf("find client: %w", err)
	}

	return ConsentView{
		ClientName:   client.Name,
		RedirectHost: domain.RedirectHost(request.RedirectURI),
		Scopes:       s.ScopeInfo(request.Resource, request.Scopes),
	}, nil
}

// ScopeInfo attaches plain-language descriptions to scope ids for a resource.
func (s *Service) ScopeInfo(resourceURI string, ids []string) []domain.Scope {
	resource, _ := s.Resources.Find(resourceURI)
	known := scopeDescriptions(resource)
	scopes := make([]domain.Scope, 0, len(ids))

	for _, id := range ids {
		scopes = append(scopes, domain.Scope{ID: id, Description: known[id]})
	}

	return scopes
}

func (s *Service) pending(ctx context.Context, id string, now time.Time) (domain.AuthorizationRequest, error) {
	request, err := s.Requests.FindRequest(ctx, id, now)
	if err != nil {
		return domain.AuthorizationRequest{}, fmt.Errorf("find request: %w", err)
	}

	return request, nil
}
