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
