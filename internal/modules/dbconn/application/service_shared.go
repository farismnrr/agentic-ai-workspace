package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

const compensationTimeout = 10 * time.Second

// ProvisionShared creates or reconciles the project's shared PostgreSQL login.
func (s *Service) ProvisionShared(ctx context.Context, userID, projectID string) (domain.Metadata, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return domain.Metadata{}, err
	}
	if configured, err := s.SharedConfigured(ctx); err != nil {
		return domain.Metadata{}, fmt.Errorf("check shared database configuration: %w", err)
	} else if !configured {
		return domain.Metadata{}, ProvisionError{Category: AdminNotConfigured}
	}
	if !s.beginProvision(projectID) {
		return domain.Metadata{}, ProvisionError{Category: ProvisionInProgress}
	}
	defer s.endProvision(projectID)

	previous, err := s.provisionSnapshot(ctx, projectID)
	if err != nil {
		return domain.Metadata{}, safeProvisionError(err)
	}
	if previous.exists && !previous.connection.Connection.Managed {
		return domain.Metadata{}, ProvisionError{Category: ManualConnectionExists}
	}
	result, err := s.provisioner.Provision(ctx, projectID)
	if err != nil {
		return domain.Metadata{}, safeProvisionError(err)
	}
	return s.saveProvisioned(ctx, projectID, result, previous)
}

type provisionSnapshot struct {
	oldPassword string
	oldEnv      string
	connection  StoredConnection
	restore     bool
	hadEnv      bool
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
	oldEnv, hadEnv := values[domain.DefaultEnvName(domain.Postgres)]
	if !exists && hadEnv {
		return provisionSnapshot{}, ProvisionError{Category: EnvironmentNameInUse}
	}
	return provisionSnapshot{
		connection: old, exists: exists, oldPassword: oldPassword, restore: restore,
		oldEnv: oldEnv, hadEnv: hadEnv,
	}, nil
}

func (s *Service) saveProvisioned(
	ctx context.Context, projectID string, result ProvisionResult, previous provisionSnapshot,
) (domain.Metadata, error) {
	conn := domain.Connection{
		Kind: domain.Postgres, Host: result.Host, Port: result.Port, Database: result.Database,
		User: result.Role, Password: result.Password, TLS: result.TLS,
		EnvName: domain.DefaultEnvName(domain.Postgres), Managed: true,
	}
	if validateErr := conn.Validate(); validateErr != nil {
		if failed, remote := s.compensate(ctx, projectID, previous, conn.EnvName); failed {
			return domain.Metadata{}, ProvisionError{Category: CompensationFailed, Remote: remote}
		}
		return domain.Metadata{}, ProvisionError{Category: ProvisionFailed}
	}
	sealed, err := s.cipher.Encrypt([]byte(conn.Password))
	if err != nil {
		if failed, remote := s.compensate(ctx, projectID, previous, conn.EnvName); failed {
			return domain.Metadata{}, ProvisionError{Category: CompensationFailed, Remote: remote}
		}
		return domain.Metadata{}, ProvisionError{Category: LocalSaveFailed}
	}
	storedConnection := conn
	storedConnection.Password = ""
	if err := s.store.Save(ctx, projectID, StoredConnection{
		Connection: storedConnection, PasswordSealed: sealed,
	}); err != nil {
		if failed, remote := s.compensate(ctx, projectID, previous, conn.EnvName); failed {
			return domain.Metadata{}, ProvisionError{Category: CompensationFailed, Remote: remote}
		}
		return domain.Metadata{}, ProvisionError{Category: LocalSaveFailed}
	}
	if err := s.env.SetSecret(ctx, projectID, conn.EnvName, conn.URL()); err != nil {
		if failed, remote := s.compensate(ctx, projectID, previous, conn.EnvName); failed {
			return domain.Metadata{}, ProvisionError{Category: CompensationFailed, Remote: remote}
		}
		return domain.Metadata{}, ProvisionError{Category: EnvironmentWriteFailed}
	}
	return conn.Meta(), nil
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

func (s *Service) compensate(
	ctx context.Context, projectID string, previous provisionSnapshot, envName string,
) (bool, bool) {
	compensationCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), compensationTimeout)
	defer cancel()
	failed := false
	remoteFailed := false
	if previous.exists {
		if err := s.store.Save(compensationCtx, projectID, previous.connection); err != nil {
			failed = true
		}
	} else if err := s.store.Delete(compensationCtx, projectID); err != nil {
		failed = true
	}
	if previous.hadEnv {
		if err := s.env.SetSecret(compensationCtx, projectID, envName, previous.oldEnv); err != nil {
			failed = true
		}
	} else if err := s.env.DeleteSecret(compensationCtx, projectID, envName); err != nil {
		failed = true
	}
	if previous.restore {
		if err := s.provisioner.Restore(compensationCtx, projectID, previous.oldPassword); err != nil {
			failed = true
			remoteFailed = true
		}
	}
	return failed, remoteFailed
}

func safeProvisionError(err error) error {
	var failure ProvisionError
	if errors.As(err, &failure) {
		return failure
	}
	return ProvisionError{Category: ProvisionFailed}
}

// SharedConfigured reports whether shared PostgreSQL provisioning is available.
func (s *Service) SharedConfigured(ctx context.Context) (bool, error) {
	if s == nil || s.provisioner == nil {
		return false, nil
	}
	configured, err := s.provisioner.Configured(ctx)
	if err != nil {
		return false, fmt.Errorf("check provisioner configuration: %w", err)
	}
	return configured, nil
}
