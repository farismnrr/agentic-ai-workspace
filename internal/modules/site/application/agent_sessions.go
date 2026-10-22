package application

import (
	"context"
	"fmt"
	"net/http"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// AgentSessionScope is resolved by server code for one task container. It must never come from model input.
type AgentSessionScope struct {
	UserID      string
	ProjectID   string
	TaskID      string
	ContainerID string
}

// AgentSessions keeps isolated browser cookies for bounded, serialized operations.
type AgentSessions interface {
	Do(ctx context.Context, scope AgentSessionScope, operation func(context.Context, http.CookieJar) error) error
	Reset(scope AgentSessionScope)
	RevokeTask(userID, taskID string)
	RevokeProject(projectID string)
	RevokeUser(userID string)
	Close()
}

// ResetAgentSession clears only this task's cookie jar. The operation is resolved against the current site.
func (s *Service) ResetAgentSession(ctx context.Context, userID, projectID, taskID string) error {
	site, err := s.deps.Store.ByProject(ctx, projectID)
	if err != nil || site.UserID != userID {
		return domain.ErrNotRunning
	}
	if s.deps.AgentSessions == nil || userID == "" || taskID == "" {
		return domain.ErrNoPicture
	}
	s.deps.AgentSessions.RevokeTask(userID, taskID)
	return nil
}

func (s *Service) withAgentSession(
	ctx context.Context, site domain.Site, userID, projectID, taskID string,
	operation func(context.Context, http.CookieJar) error,
) error {
	if s.deps.AgentSessions == nil || userID == "" || projectID == "" || taskID == "" || operation == nil {
		return domain.ErrNoPicture
	}
	scope := AgentSessionScope{UserID: userID, ProjectID: projectID, TaskID: taskID, ContainerID: site.ContainerID}
	if err := s.deps.AgentSessions.Do(ctx, scope, operation); err != nil {
		return fmt.Errorf("use agent session: %w", err)
	}
	return nil
}
