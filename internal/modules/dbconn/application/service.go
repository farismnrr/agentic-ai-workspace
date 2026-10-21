package application

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var ErrNotFound = domain.ErrMissing

type Service struct {
	store        Store
	cipher       Cipher
	owner        ProjectOwner
	env          Environment
	probe        Prober
	provisioner  Provisioner
	provisioning map[string]bool
	provisionMu  sync.Mutex
}

func NewService(
	store Store, cipher Cipher, owner ProjectOwner, env Environment, probe Prober,
	provisioners ...Provisioner,
) *Service {
	service := &Service{store: store, cipher: cipher, owner: owner, env: env, probe: probe}
	if len(provisioners) > 0 {
		service.provisioner = provisioners[0]
	}
	return service
}

func (s *Service) Get(ctx context.Context, userID, projectID string) (domain.Metadata, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return domain.Metadata{}, err
	}
	stored, err := s.store.Get(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		return domain.Metadata{}, nil
	}
	if err != nil {
		return domain.Metadata{}, fmt.Errorf("load database connection: %w", err)
	}
	return stored.Connection.Meta(), nil
}

func (s *Service) previousConnection(ctx context.Context, projectID string) (StoredConnection, bool, error) {
	old, err := s.store.Get(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		return StoredConnection{}, false, nil
	}
	if err != nil {
		return StoredConnection{}, false, fmt.Errorf("load previous database connection: %w", err)
	}
	return old, true, nil
}

func (s *Service) Delete(ctx context.Context, userID, projectID string) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}
	if !s.beginProvision(projectID) {
		return ProvisionError{Category: ProvisionInProgress}
	}
	defer s.endProvision(projectID)
	stored, err := s.store.Get(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load database connection for delete: %w", err)
	}
	if err := s.store.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("delete database connection: %w", err)
	}
	if err := s.env.DeleteSecret(ctx, projectID, stored.Connection.EnvName); err != nil {
		_ = s.store.Save(ctx, projectID, stored)
		return fmt.Errorf("remove database environment variable: %w", err)
	}
	return nil
}

func (s *Service) Test(ctx context.Context, userID, projectID string, submitted *domain.Connection) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}
	conn, err := s.testConnection(ctx, projectID, submitted)
	if err != nil {
		return err
	}
	if err := conn.Validate(); err != nil {
		return fmt.Errorf("validate database connection: %w", err)
	}
	if err := s.probe.Probe(ctx, conn); err != nil {
		return fmt.Errorf("probe database connection: %w", err)
	}
	return nil
}

func (s *Service) testConnection(
	ctx context.Context, projectID string, submitted *domain.Connection,
) (domain.Connection, error) {
	if submitted != nil {
		conn := *submitted
		if conn.EnvName == "" {
			conn.EnvName = domain.DefaultEnvName(conn.Kind)
		}
		return conn, nil
	}
	stored, err := s.store.Get(ctx, projectID)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("load database connection for test: %w", err)
	}
	password, err := s.cipher.Decrypt(stored.PasswordSealed)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("open database password: %w", err)
	}
	conn := stored.Connection
	conn.Password = string(password)
	return conn, nil
}

func (s *Service) checkOwner(ctx context.Context, userID, projectID string) error {
	owner, _, err := s.owner.Owner(ctx, projectID)
	if err != nil || owner != userID {
		return ErrNotFound
	}
	return nil
}
