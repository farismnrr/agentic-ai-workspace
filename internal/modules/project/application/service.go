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
	Find(ctx context.Context, userID, id string) (domain.Project, error)
	Update(ctx context.Context, userID, id, name, description string) error
	Archive(ctx context.Context, userID, id string, at time.Time) error
}

// Folders creates the folder that holds a project's files.
type Folders interface {
	Create(id string) error
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}

// Service holds the project use cases.
type Service struct {
	store   Store
	folders Folders
	ids     IDs
	now     func() time.Time
}

// NewService wires the use cases.
func NewService(store Store, folders Folders, ids IDs, now func() time.Time) *Service {
	return &Service{store: store, folders: folders, ids: ids, now: now}
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
