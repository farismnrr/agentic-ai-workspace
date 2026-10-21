package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

// DetachSQLite removes the project connection and generated DATABASE_URL only.
func (s *Service) DetachSQLite(ctx context.Context, userID, projectID string) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}
	if !s.beginProvision(projectID) {
		return ProvisionError{Category: ProvisionInProgress}
	}
	defer s.endProvision(projectID)
	stored, err := s.store.Get(ctx, projectID)
	exists := err == nil
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("load database connection for detach: %w", err)
	}
	if exists {
		if err := s.store.Delete(ctx, projectID); err != nil {
			return fmt.Errorf("delete database connection: %w", err)
		}
	}
	if exists && stored.Connection.Managed {
		if err := s.env.DeleteSecret(ctx, projectID, domain.DefaultEnvName(domain.Postgres)); err != nil {
			_ = s.store.Save(ctx, projectID, stored)
			return fmt.Errorf("remove database environment variable: %w", err)
		}
	}
	return nil
}
