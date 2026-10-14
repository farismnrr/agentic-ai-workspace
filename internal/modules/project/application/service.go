package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// Store persists projects.
type Store interface {
	Create(ctx context.Context, p domain.Project) error
	List(ctx context.Context, userID string) ([]domain.Project, error)
	ListAll(ctx context.Context, userID string) ([]domain.Project, error)
	Find(ctx context.Context, userID, id string) (domain.Project, error)
	Update(ctx context.Context, userID, id, name, description string) error
	Archive(ctx context.Context, userID, id string, at time.Time) error
	// DeleteAll removes every project a person has, archived ones too, and returns their ids.
	DeleteAll(ctx context.Context, userID string) ([]string, error)
}

// Folders creates the folder that holds a project's files.
type Folders interface {
	Create(id string) error
	Remove(id string) error
}

// Check is a named command the AI may run to prove its work in a project.
type Check struct {
	Name           string
	Description    string
	Command        []string
	TimeoutSeconds int
}

// Checks reads and writes the checks stored in a project's folder. Save must
// wrap domain.ErrInvalidChecks when the checks are not acceptable.
type Checks interface {
	List(ctx context.Context, projectID string) ([]Check, error)
	Save(ctx context.Context, projectID string, checks []Check) error
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}

// Service holds the project use cases.
type Service struct {
	store   Store
	folders Folders
	checks  Checks
	ids     IDs
	now     func() time.Time
}

// NewService wires the use cases.
func NewService(store Store, folders Folders, checks Checks, ids IDs, now func() time.Time) *Service {
	return &Service{store: store, folders: folders, checks: checks, ids: ids, now: now}
}

// Create makes a project and its folder.
func (s *Service) Create(ctx context.Context, userID, name, description string) (domain.Project, error) {
	name, description, err := domain.Normalize(name, description)
	if err != nil {
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}

	id, err := s.ids.ID()
	if err != nil {
		return domain.Project{}, fmt.Errorf("new project id: %w", err)
	}

	if err := s.folders.Create(id); err != nil {
		return domain.Project{}, fmt.Errorf("create folder: %w", err)
	}

	project := domain.Project{ID: id, UserID: userID, Name: name, Description: description, CreatedAt: s.now()}
	if err := s.store.Create(ctx, project); err != nil {
		return domain.Project{}, fmt.Errorf("save project: %w", err)
	}

	return project, nil
}

// List returns a person's active projects, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]domain.Project, error) {
	projects, err := s.store.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}

// Get returns one of the person's projects.
func (s *Service) Get(ctx context.Context, userID, id string) (domain.Project, error) {
	project, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return domain.Project{}, fmt.Errorf("find project: %w", err)
	}

	return project, nil
}

// Update renames a project or changes its description.
func (s *Service) Update(ctx context.Context, userID, id, name, description string) (domain.Project, error) {
	name, description, err := domain.Normalize(name, description)
	if err != nil {
		return domain.Project{}, fmt.Errorf("update project: %w", err)
	}

	if err := s.store.Update(ctx, userID, id, name, description); err != nil {
		return domain.Project{}, fmt.Errorf("update project: %w", err)
	}

	return s.Get(ctx, userID, id)
}

// Archive hides a project; its folder and tasks are kept.
func (s *Service) Archive(ctx context.Context, userID, id string) error {
	if err := s.store.Archive(ctx, userID, id, s.now()); err != nil {
		return fmt.Errorf("archive project: %w", err)
	}

	return nil
}

// Checks returns the commands the AI may run in one of the person's projects.
func (s *Service) Checks(ctx context.Context, userID, id string) ([]Check, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return nil, err
	}

	checks, err := s.checks.List(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}

	return checks, nil
}

// SaveChecks replaces the commands the AI may run in a project.
func (s *Service) SaveChecks(ctx context.Context, userID, id string, checks []Check) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}

	if err := s.checks.Save(ctx, id, checks); err != nil {
		return fmt.Errorf("save checks: %w", err)
	}

	return nil
}
