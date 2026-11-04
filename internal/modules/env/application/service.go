package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
)

// StoredVariable is a database record; ciphertext never crosses the application boundary to a caller.
type StoredVariable struct {
	UpdatedAt time.Time
	Name      string
	Sealed    string
	IsSecret  bool
}

// Store persists sealed project variables.
type Store interface {
	List(ctx context.Context, projectID string) ([]StoredVariable, error)
	Put(ctx context.Context, projectID string, variable StoredVariable) error
	Delete(ctx context.Context, projectID, name string) error
}

// Cipher seals and opens variable values.
type Cipher interface {
	Encrypt([]byte) (string, error)
	Decrypt(string) ([]byte, error)
}

// ProjectOwner resolves the owner of an active project.
type ProjectOwner interface {
	Owner(ctx context.Context, projectID string) (userID, slug string, err error)
}

// PasswordVerifier validates a current account password for a sensitive action.
type PasswordVerifier interface {
	VerifyPassword(ctx context.Context, userID, password string) (bool, error)
}

// RevealAudit records metadata only for a browser secret reveal attempt.
type RevealAudit interface {
	RecordReveal(ctx context.Context, userID, projectID, name, outcome string, at time.Time) error
}

// Service holds project-scoped Env use cases.
type Service struct {
	store  Store
	cipher Cipher
	owner  ProjectOwner
	now    func() time.Time
}

// NewService wires the Env use cases.
func NewService(store Store, cipher Cipher, owner ProjectOwner, now func() time.Time) *Service {
	return &Service{store: store, cipher: cipher, owner: owner, now: now}
}

// List returns variables for a project after confirming the caller owns it.
func (s *Service) List(ctx context.Context, userID, projectID string) ([]domain.Variable, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return nil, err
	}

	stored, err := s.store.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project variables: %w", err)
	}

	return s.openVariables(stored, false)
}

// Put creates or replaces one variable. Secret values are never returned.
func (s *Service) Put(
	ctx context.Context, userID, projectID, name, value string, isSecret bool,
) (domain.Variable, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return domain.Variable{}, err
	}
	if err := domain.ValidateName(name); err != nil {
		return domain.Variable{}, fmt.Errorf("validate variable name: %w", err)
	}
	if err := domain.ValidateValue(value); err != nil {
		return domain.Variable{}, fmt.Errorf("validate variable value: %w", err)
	}

	sealed, err := s.cipher.Encrypt([]byte(value))
	if err != nil {
		return domain.Variable{}, fmt.Errorf("seal project variable: %w", err)
	}

	updatedAt := time.Unix(s.now().Unix(), 0).UTC()
	variable := StoredVariable{Name: name, Sealed: sealed, IsSecret: isSecret, UpdatedAt: updatedAt}
	if err := s.store.Put(ctx, projectID, variable); err != nil {
		return domain.Variable{}, fmt.Errorf("save project variable: %w", err)
	}

	if isSecret {
		value = ""
	}

	return domain.Variable{Name: name, Value: value, IsSecret: isSecret, UpdatedAt: variable.UpdatedAt}, nil
}

// Delete removes one variable after confirming the caller owns its project.
func (s *Service) Delete(ctx context.Context, userID, projectID, name string) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}
	if err := domain.ValidateName(name); err != nil {
		return fmt.Errorf("validate variable name: %w", err)
	}

	if err := s.store.Delete(ctx, projectID, name); err != nil {
		return fmt.Errorf("delete project variable: %w", err)
	}
	return nil
}

// SetSecret writes a server-generated secret for another application feature.
func (s *Service) SetSecret(ctx context.Context, projectID, name, value string) error {
	if err := domain.ValidateName(name); err != nil {
		return fmt.Errorf("validate variable name: %w", err)
	}
	if err := domain.ValidateSecretValue(value); err != nil {
		return fmt.Errorf("validate variable value: %w", err)
	}
	sealed, err := s.cipher.Encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("seal project variable: %w", err)
	}
	if err := s.store.Put(ctx, projectID, StoredVariable{
		Name: name, Sealed: sealed, IsSecret: true, UpdatedAt: time.Unix(s.now().Unix(), 0).UTC(),
	}); err != nil {
		return fmt.Errorf("save project secret: %w", err)
	}
	return nil
}

// DeleteSecret removes a feature-owned secret variable from a project.
func (s *Service) DeleteSecret(ctx context.Context, projectID, name string) error {
	if err := domain.ValidateName(name); err != nil {
		return fmt.Errorf("validate variable name: %w", err)
	}
	err := s.store.Delete(ctx, projectID, name)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete project secret: %w", err)
	}
	return nil
}
