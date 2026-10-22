package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
)

// Secret resolves one named secret only after confirming project ownership.
func (s *Service) Secret(ctx context.Context, userID, projectID, name string) (string, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return "", err
	}
	if err := domain.ValidateName(name); err != nil {
		return "", domain.ErrNotFound
	}
	stored, err := s.store.List(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("load project secret: %w", err)
	}
	for _, variable := range stored {
		if variable.Name != name || !variable.IsSecret {
			continue
		}
		value, openErr := s.cipher.Decrypt(variable.Sealed)
		if openErr != nil {
			return "", fmt.Errorf("open project secret: %w", openErr)
		}
		return string(value), nil
	}
	return "", domain.ErrNotFound
}
