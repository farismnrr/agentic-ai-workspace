package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
)

// Values returns the decrypted name-to-value map for trusted sandbox injection.
func (s *Service) Values(ctx context.Context, projectID string) (map[string]string, error) {
	stored, err := s.store.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load project variables: %w", err)
	}

	variables, err := s.openVariables(stored, true)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, len(variables))
	for _, variable := range variables {
		values[variable.Name] = variable.Value
	}

	return values, nil
}

func (s *Service) checkOwner(ctx context.Context, userID, projectID string) error {
	owner, _, err := s.owner.Owner(ctx, projectID)
	if err != nil || owner != userID {
		return domain.ErrNotFound
	}

	return nil
}

// AuthorizeReveal confirms ownership without decrypting the selected value.
func (s *Service) AuthorizeReveal(ctx context.Context, userID, projectID string) error {
	return s.checkOwner(ctx, userID, projectID)
}

// Reveal opens one named secret after the caller has verified the current password.
func (s *Service) Reveal(ctx context.Context, userID, projectID, name string) (string, error) {
	return s.Secret(ctx, userID, projectID, name)
}

func (s *Service) openVariables(stored []StoredVariable, includeSecrets bool) ([]domain.Variable, error) {
	variables := make([]domain.Variable, 0, len(stored))
	for _, item := range stored {
		variable := domain.Variable{Name: item.Name, IsSecret: item.IsSecret, UpdatedAt: item.UpdatedAt}
		if includeSecrets || !item.IsSecret {
			value, err := s.cipher.Decrypt(item.Sealed)
			if err != nil {
				return nil, fmt.Errorf("open project variable: %w", err)
			}
			variable.Value = string(value)
		}
		variables = append(variables, variable)
	}

	return variables, nil
}
