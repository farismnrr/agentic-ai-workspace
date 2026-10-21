package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

const compensationTimeout = 10 * time.Second

type provisionSnapshot struct {
	oldEnv      map[string]string
	hadEnv      map[string]bool
	oldPassword string
	connection  StoredConnection
	restore     bool
	exists      bool
}

func (s *Service) provisionSnapshot(ctx context.Context, projectID string) (provisionSnapshot, error) {
	old, exists, err := s.previousConnection(ctx, projectID)
	if err != nil {
		return provisionSnapshot{}, err
	}
	values, err := s.env.Values(ctx, projectID)
	if err != nil {
		return provisionSnapshot{}, fmt.Errorf("read previous project environment: %w", err)
	}
	oldPassword, restore, err := s.previousManagedPassword(old, exists)
	if err != nil {
		return provisionSnapshot{}, fmt.Errorf("read previous managed password: %w", err)
	}
	envNames := managedEnvNames()
	oldEnv, hadEnv := make(map[string]string), make(map[string]bool)
	for _, name := range envNames {
		oldEnv[name], hadEnv[name] = values[name]
	}
	if !exists || !old.Connection.Managed {
		for _, name := range []string{
			"DATABASE_URL", "MONGODB_URL", "DATABASE_SSL_CA", "DATABASE_SSL_CERT", "DATABASE_SSL_KEY",
		} {
			if hadEnv[name] {
				return provisionSnapshot{}, ProvisionError{Category: EnvironmentNameInUse}
			}
		}
	}
	return provisionSnapshot{
		connection: old, exists: exists, oldPassword: oldPassword, restore: restore,
		oldEnv: oldEnv, hadEnv: hadEnv,
	}, nil
}

func (s *Service) saveProvisioned(
	ctx context.Context, projectID string, kind domain.Kind, result ProvisionResult, previous provisionSnapshot,
) (domain.Metadata, error) {
	conn := domain.Connection{
		Kind: kind, Host: result.Host, Port: result.Port, Database: result.Database,
		User: result.Role, Password: result.Password, TLS: result.TLS,
		EnvName: domain.DefaultEnvName(kind), Managed: true,
	}
	if validateErr := conn.Validate(); validateErr != nil {
		return domain.Metadata{}, s.provisionSaveError(
			ctx, projectID, previous, conn.EnvName, ProvisionFailed,
		)
	}
	sealed, err := s.cipher.Encrypt([]byte(conn.Password))
	if err != nil {
		return domain.Metadata{}, s.provisionSaveError(
			ctx, projectID, previous, conn.EnvName, LocalSaveFailed,
		)
	}
	storedConnection := conn
	storedConnection.Password = ""
	if err := s.store.Save(ctx, projectID, StoredConnection{
		Connection: storedConnection, PasswordSealed: sealed,
	}); err != nil {
		return domain.Metadata{}, s.provisionSaveError(
			ctx, projectID, previous, conn.EnvName, LocalSaveFailed,
		)
	}
	previousConnection := previous.connection.Connection
	previousManaged := previous.exists && previousConnection.Managed
	if err := s.writeProvisionEnvironment(
		ctx, projectID, conn, result, previousManaged, previousConnection.EnvName,
	); err != nil {
		return domain.Metadata{}, s.provisionSaveError(
			ctx, projectID, previous, conn.EnvName, EnvironmentWriteFailed,
		)
	}
	metadata := conn.Meta()
	metadata.Choice = string(kind)
	return metadata, nil
}

func (s *Service) provisionSaveError(
	ctx context.Context, projectID string, previous provisionSnapshot, envName string,
	category ProvisionFailure,
) error {
	if failed, remote := s.compensate(ctx, projectID, previous, envName); failed {
		return ProvisionError{Category: CompensationFailed, Remote: remote}
	}
	return ProvisionError{Category: category}
}

func (s *Service) beginProvision(projectID string) bool {
	s.provisionMu.Lock()
	defer s.provisionMu.Unlock()
	if s.provisioning == nil {
		s.provisioning = make(map[string]bool)
	}
	if s.provisioning[projectID] {
		return false
	}
	s.provisioning[projectID] = true
	return true
}

func (s *Service) endProvision(projectID string) {
	s.provisionMu.Lock()
	delete(s.provisioning, projectID)
	s.provisionMu.Unlock()
}

func (s *Service) previousManagedPassword(old StoredConnection, exists bool) (string, bool, error) {
	if !exists || !old.Connection.Managed {
		return "", false, nil
	}
	password, err := s.cipher.Decrypt(old.PasswordSealed)
	if err != nil {
		return "", false, fmt.Errorf("decrypt previous managed password: %w", err)
	}
	return string(password), true, nil
}

func safeProvisionError(err error) error {
	var failure ProvisionError
	if errors.As(err, &failure) {
		return failure
	}
	return ProvisionError{Category: ProvisionFailed}
}

// SharedConfigured reports whether PostgreSQL provisioning is available.
func (s *Service) SharedConfigured(ctx context.Context) (bool, error) {
	provisioner := s.provisioner(domain.Postgres)
	if provisioner == nil {
		return false, nil
	}
	configured, err := provisioner.Configured(ctx)
	if err != nil {
		return false, fmt.Errorf("check provisioner configuration: %w", err)
	}
	return configured, nil
}
