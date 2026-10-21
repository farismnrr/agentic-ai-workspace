package application

import (
	"context"
	"errors"
	"fmt"
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
		return s.removeManagedEnvironment(ctx, projectID, stored)
	}
	return nil
}

func (s *Service) removeManagedEnvironment(ctx context.Context, projectID string, stored StoredConnection) error {
	values, err := s.env.Values(ctx, projectID)
	if err != nil {
		_ = s.store.Save(ctx, projectID, stored)
		return fmt.Errorf("read database environment variables: %w", err)
	}
	for _, name := range []string{stored.Connection.EnvName, "DATABASE_SSL_CA", "DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if err := s.env.DeleteSecret(ctx, projectID, name); err != nil {
			_ = s.store.Save(ctx, projectID, stored)
			restoreManagedEnvironment(ctx, s.env, projectID, values)
			return fmt.Errorf("remove database environment variable: %w", err)
		}
	}
	return nil
}

func restoreManagedEnvironment(ctx context.Context, env Environment, projectID string, values map[string]string) {
	for _, name := range managedEnvNames() {
		if value, exists := values[name]; exists {
			_ = env.SetSecret(ctx, projectID, name, value)
		}
	}
}
