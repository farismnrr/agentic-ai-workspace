package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

const (
	maxRedirectURIs = 10
	maxNameLength   = 100
	defaultAppName  = "Unnamed app"
)

// Registration is a Dynamic Client Registration request (RFC 7591).
type Registration struct {
	Name         string
	AuthMethod   string
	RedirectURIs []string
}

// ResolveClient finds a client by id. A URL id is a Client ID Metadata
// Document: it is re-fetched so changes to its redirect URIs apply, falling
// back to the last stored copy when the document is unreachable.
func (s *Service) ResolveClient(ctx context.Context, id string) (domain.Client, error) {
	if domain.IsMetadataClientID(id) {
		return s.resolveMetadata(ctx, id)
	}

	client, err := s.Clients.FindClient(ctx, id)
	if errors.Is(err, domain.ErrClientNotFound) {
		return domain.Client{}, domain.NewError(domain.CodeInvalidClient, "unknown client_id")
	}

	if err != nil {
		return domain.Client{}, fmt.Errorf("find client: %w", err)
	}

	return client, nil
}

func (s *Service) resolveMetadata(ctx context.Context, id string) (domain.Client, error) {
	fetched, fetchErr := s.Metadata.Fetch(ctx, id)
	if fetchErr == nil {
		if err := s.Clients.SaveClient(ctx, fetched); err != nil {
			return domain.Client{}, fmt.Errorf("save client: %w", err)
		}

		return fetched, nil
	}

	stored, err := s.Clients.FindClient(ctx, id)
	if err != nil {
		return domain.Client{}, domain.NewError(domain.CodeInvalidClient, "client metadata document is not valid")
	}

	return stored, nil
}

// RegisteredClient is the result of a DCR registration (RFC 7591).
type RegisteredClient struct {
	ClientSecret string
	Client       domain.Client
}

func (s *Service) authenticateClient(ctx context.Context, clientID, clientSecret string) error {
	client, err := s.ResolveClient(ctx, clientID)
	if err != nil {
		return err
	}

	if client.IsConfidential() && !client.VerifySecret(clientSecret, s.Tokens.Hash) {
		return domain.NewError(domain.CodeInvalidClient, "client authentication failed")
	}

	return nil
}

// RegisterClient creates a client from a DCR request.
func (s *Service) RegisterClient(ctx context.Context, reg Registration) (RegisteredClient, error) {
	authMethod, authErr := validateAuthMethod(reg.AuthMethod)
	if authErr != nil {
		return RegisteredClient{}, authErr
	}

	if uriErr := validateRedirectURIs(reg.RedirectURIs); uriErr != nil {
		return RegisteredClient{}, uriErr
	}

	id, idErr := s.Tokens.ID()
	if idErr != nil {
		return RegisteredClient{}, fmt.Errorf("new client id: %w", idErr)
	}

	plainSecret, hashSecret, secErr := s.generateSecret(authMethod)
	if secErr != nil {
		return RegisteredClient{}, secErr
	}

	client := domain.Client{
		ID: id, Name: clientName(reg.Name), Source: domain.SourceDCR,
		AuthMethod: authMethod, SecretHash: hashSecret, RedirectURIs: reg.RedirectURIs,
	}
	if err := s.Clients.SaveClient(ctx, client); err != nil {
		return RegisteredClient{}, fmt.Errorf("save client: %w", err)
	}

	return RegisteredClient{Client: client, ClientSecret: plainSecret}, nil
}

func (s *Service) generateSecret(authMethod string) (plain, hash string, err error) {
	if authMethod == domain.AuthMethodNone {
		return "", "", nil
	}

	p, h, genErr := s.Tokens.New()
	if genErr != nil {
		return "", "", fmt.Errorf("new client secret: %w", genErr)
	}

	return p, h, nil
}

func validateAuthMethod(method string) (string, error) {
	if method == "" {
		return domain.AuthMethodClientSecretBasic, nil
	}

	switch method {
	case domain.AuthMethodNone, domain.AuthMethodClientSecretBasic, domain.AuthMethodClientSecretPost:
		return method, nil
	default:
		return "", domain.NewError(domain.CodeInvalidClientMetadata, "unsupported token_endpoint_auth_method")
	}
}

func validateRedirectURIs(uris []string) error {
	if len(uris) == 0 || len(uris) > maxRedirectURIs {
		return domain.NewError(domain.CodeInvalidRedirectURI, "provide 1 to 10 redirect_uris")
	}

	for _, uri := range uris {
		if err := domain.ValidateRedirectURI(uri); err != nil {
			return domain.NewError(domain.CodeInvalidRedirectURI, err.Error())
		}
	}

	return nil
}

func clientName(raw string) string {
	runes := []rune(raw)
	if len(runes) == 0 {
		return defaultAppName
	}

	if len(runes) > maxNameLength {
		runes = runes[:maxNameLength]
	}

	return string(runes)
}
