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

// RegisterClient creates a public client from a DCR request.
func (s *Service) RegisterClient(ctx context.Context, reg Registration) (domain.Client, error) {
	if reg.AuthMethod != "" && reg.AuthMethod != "none" {
		return domain.Client{}, domain.NewError(domain.CodeInvalidClientMetadata, "only public clients are supported")
	}

	if len(reg.RedirectURIs) == 0 || len(reg.RedirectURIs) > maxRedirectURIs {
		return domain.Client{}, domain.NewError(domain.CodeInvalidRedirectURI, "provide 1 to 10 redirect_uris")
	}

	for _, uri := range reg.RedirectURIs {
		if err := domain.ValidateRedirectURI(uri); err != nil {
			return domain.Client{}, domain.NewError(domain.CodeInvalidRedirectURI, err.Error())
		}
	}

	id, err := s.Tokens.ID()
	if err != nil {
		return domain.Client{}, fmt.Errorf("new client id: %w", err)
	}

	client := domain.Client{ID: id, Name: clientName(reg.Name), RedirectURIs: reg.RedirectURIs, Source: domain.SourceDCR}
	if err := s.Clients.SaveClient(ctx, client); err != nil {
		return domain.Client{}, fmt.Errorf("save client: %w", err)
	}

	return client, nil
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
