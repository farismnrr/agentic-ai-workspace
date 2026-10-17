package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const maxCommandLength = 4000

// Deps are the collaborators of Service.
type Deps struct {
	Store    Store
	Runtime  Runtime
	Projects Projects
	Notify   Notifier
	IDs      IDs
	Now      func() time.Time
	Secret   []byte
	Limits   Limits
}

// Service starts, stops and lists sites.
type Service struct{ deps Deps }

// NewService wires the use cases.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// Start runs the project's app and returns its site. A project has one site; starting it
// again replaces the running app and keeps its address.
func (s *Service) Start(ctx context.Context, projectID, command string, port int) (domain.Site, error) {
	command, err := checkStart(command, port)
	if err != nil {
		return domain.Site{}, err
	}

	userID, name, err := s.deps.Projects.Owner(ctx, projectID)
	if err != nil {
		return domain.Site{}, fmt.Errorf("start site: %w", err)
	}

	// Room is checked first: a refused start must not stop the app the project already runs.
	if err = s.checkRoom(ctx, userID, projectID); err != nil {
		return domain.Site{}, err
	}

	previous, err := s.stopExisting(ctx, projectID)
	if err != nil {
		return domain.Site{}, err
	}

	token := previous.Token
	if token == "" {
		if token, err = s.deps.IDs.UUID(); err != nil {
			return domain.Site{}, fmt.Errorf("make site address: %w", err)
		}
	}

	site := domain.Site{
		ProjectID: projectID, UserID: userID, Slug: domain.Slugify(name), Token: token, Command: command,
		Visibility: domain.Private, Status: domain.Running, Port: port, StartedAt: s.deps.Now(),
	}
	if previous.Visibility != "" {
		site.Visibility = previous.Visibility
	}

	site.ContainerID, err = s.deps.Runtime.Start(ctx, StartSpec{
		ProjectID: projectID, BasePath: site.BasePath(), Command: command, Port: port,
	})
	if err != nil {
		return domain.Site{}, fmt.Errorf("start site: %w", err)
	}

	if err = s.deps.Store.Save(ctx, site); err != nil {
		s.deps.Runtime.Stop(ctx, site.ContainerID)

		return domain.Site{}, fmt.Errorf("save site: %w", err)
	}

	s.deps.Notify.Changed(userID)

	return site, nil
}

// checkStart validates what the agent asked for and returns the trimmed command.
func checkStart(command string, port int) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" || len(command) > maxCommandLength {
		return "", domain.ErrBadCommand
	}

	if err := domain.ValidPort(port); err != nil {
		return "", fmt.Errorf("start site: %w", err)
	}

	return command, nil
}

// stopExisting stops the project's running app, if any, and returns the old record.
func (s *Service) stopExisting(ctx context.Context, projectID string) (domain.Site, error) {
	old, err := s.deps.Store.ByProject(ctx, projectID)
	if err != nil {
		if isNotFound(err) {
			return domain.Site{}, nil
		}

		return domain.Site{}, fmt.Errorf("find site: %w", err)
	}

	if old.Status == domain.Running {
		s.deps.Runtime.Stop(ctx, old.ContainerID)

		if err = s.deps.Store.MarkStopped(ctx, projectID); err != nil {
			return domain.Site{}, fmt.Errorf("mark site stopped: %w", err)
		}
	}

	return old, nil
}

// checkRoom refuses a start when the person or the server already runs enough sites. A project has one app, so
// its own is never counted; a limit of zero means no limit.
func (s *Service) checkRoom(ctx context.Context, userID, projectID string) error {
	running, err := s.deps.Store.Running(ctx)
	if err != nil {
		return fmt.Errorf("count sites: %w", err)
	}

	mine, total := 0, 0

	for _, other := range running {
		if other.ProjectID == projectID {
			continue
		}

		total++

		if other.UserID == userID {
			mine++
		}
	}

	switch {
	case s.deps.Limits.PerUser > 0 && mine >= s.deps.Limits.PerUser:
		return domain.ErrTooMany
	case s.deps.Limits.Total > 0 && total >= s.deps.Limits.Total:
		return domain.ErrServerBusy
	}

	return nil
}
