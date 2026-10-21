package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func managedEnvNames() []string {
	return []string{"DATABASE_URL", "MONGODB_URL", "DATABASE_SSL_CA", "DATABASE_SSL_CERT", "DATABASE_SSL_KEY"}
}

func (s *Service) writeProvisionEnvironment(
	ctx context.Context, projectID string, conn domain.Connection, result ProvisionResult,
	previousManaged bool, previousEnvName string,
) error {
	if previousManaged && previousEnvName != "" && previousEnvName != conn.EnvName {
		if err := s.env.DeleteSecret(ctx, projectID, previousEnvName); err != nil {
			return fmt.Errorf("remove previous managed database URL: %w", err)
		}
	}
	if err := s.env.SetSecret(ctx, projectID, conn.EnvName, conn.URLWithSSLMode(result.SSLMode)); err != nil {
		return fmt.Errorf("save shared database URL: %w", err)
	}
	if previousManaged {
		return s.reconcileManagedSSLEnvironment(ctx, projectID, result.CACert)
	}
	return s.writeInitialCAEnvironment(ctx, projectID, result.CACert)
}

func (s *Service) reconcileManagedSSLEnvironment(ctx context.Context, projectID, caCert string) error {
	if caCert == "" {
		if err := s.env.DeleteSecret(ctx, projectID, "DATABASE_SSL_CA"); err != nil {
			return fmt.Errorf("remove shared database CA environment: %w", err)
		}
	} else if err := s.env.SetSecret(ctx, projectID, "DATABASE_SSL_CA", caCert); err != nil {
		return fmt.Errorf("save shared database CA environment: %w", err)
	}
	return s.removeManagedClientCertificateEnvironment(ctx, projectID)
}

func (s *Service) removeManagedClientCertificateEnvironment(ctx context.Context, projectID string) error {
	for _, name := range []string{"DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if err := s.env.DeleteSecret(ctx, projectID, name); err != nil {
			return fmt.Errorf("remove shared database client certificate environment: %w", err)
		}
	}
	return nil
}

func (s *Service) writeInitialCAEnvironment(ctx context.Context, projectID, caCert string) error {
	if caCert == "" {
		return nil
	}
	if err := s.env.SetSecret(ctx, projectID, "DATABASE_SSL_CA", caCert); err != nil {
		return fmt.Errorf("save shared database CA environment: %w", err)
	}
	return nil
}

func (s *Service) compensate(
	ctx context.Context, projectID string, previous provisionSnapshot, _ string,
) (bool, bool) {
	compensationCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), compensationTimeout)
	defer cancel()
	failed := s.restoreLocalState(compensationCtx, projectID, previous)
	remoteFailed := false
	if previous.restore {
		provisioner := s.provisioner(previous.connection.Connection.Kind)
		if provisioner == nil || provisioner.Restore(compensationCtx, projectID, previous.oldPassword) != nil {
			failed, remoteFailed = true, true
		}
	}
	return failed, remoteFailed
}

func (s *Service) restoreLocalState(
	ctx context.Context, projectID string, previous provisionSnapshot,
) bool {
	connectionFailed := s.restoreConnection(ctx, projectID, previous)
	environmentFailed := s.restoreEnvironment(ctx, projectID, previous)
	return connectionFailed || environmentFailed
}

func (s *Service) restoreConnection(ctx context.Context, projectID string, previous provisionSnapshot) bool {
	if previous.exists {
		if err := s.store.Save(ctx, projectID, previous.connection); err != nil {
			return true
		}
	} else if err := s.store.Delete(ctx, projectID); err != nil {
		return true
	}
	return false
}

func (s *Service) restoreEnvironment(ctx context.Context, projectID string, previous provisionSnapshot) bool {
	failed := false
	for _, name := range managedEnvNames() {
		if previous.hadEnv[name] {
			if err := s.env.SetSecret(ctx, projectID, name, previous.oldEnv[name]); err != nil {
				failed = true
			}
		} else if err := s.env.DeleteSecret(ctx, projectID, name); err != nil {
			failed = true
		}
	}
	return failed
}
