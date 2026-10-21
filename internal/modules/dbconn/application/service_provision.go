package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

// ProvisionShared preserves the original PostgreSQL entry point for existing callers.
func (s *Service) ProvisionShared(ctx context.Context, userID, projectID string) (domain.Metadata, error) {
	return s.Provision(ctx, userID, projectID, domain.Postgres)
}

// Provision creates or reconciles an isolated managed connection for the selected kind.
func (s *Service) Provision(
	ctx context.Context, userID, projectID string, kind domain.Kind,
) (domain.Metadata, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return domain.Metadata{}, err
	}
	provisioner, err := s.configuredProvisioner(ctx, kind)
	if err != nil {
		return domain.Metadata{}, err
	}
	if !s.beginProvision(projectID) {
		return domain.Metadata{}, ProvisionError{Category: ProvisionInProgress}
	}
	defer s.endProvision(projectID)
	previous, err := s.provisionSnapshot(ctx, projectID)
	if err != nil {
		return domain.Metadata{}, safeProvisionError(err)
	}
	if switchErr := validateProvisionSwitch(previous, kind); switchErr != nil {
		return domain.Metadata{}, switchErr
	}
	result, err := provisioner.Provision(ctx, projectID)
	if err != nil {
		return domain.Metadata{}, safeProvisionError(err)
	}
	return s.saveProvisioned(ctx, projectID, kind, result, previous)
}

func (s *Service) configuredProvisioner(
	ctx context.Context, kind domain.Kind,
) (Provisioner, error) {
	provisioner := s.provisioner(kind)
	if provisioner == nil {
		return nil, ProvisionError{Category: KindNotConfigured}
	}
	configured, err := provisioner.Configured(ctx)
	if err != nil {
		return nil, fmt.Errorf("check %s database configuration: %w", kind, err)
	}
	if !configured {
		return nil, ProvisionError{Category: KindNotConfigured}
	}
	return provisioner, nil
}

func validateProvisionSwitch(previous provisionSnapshot, kind domain.Kind) error {
	if previous.exists && !previous.connection.Connection.Managed {
		return ProvisionError{Category: ManualConnectionExists}
	}
	if previous.exists && previous.connection.Connection.Managed &&
		previous.connection.Connection.EnvName != domain.DefaultEnvName(kind) &&
		previous.hadEnv[domain.DefaultEnvName(kind)] {
		return ProvisionError{Category: EnvironmentNameInUse}
	}
	return nil
}
