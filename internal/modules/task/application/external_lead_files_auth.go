package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// AuthorizeExternalFileWrite ties a project file mutation to an external-led
// task, so a connected lead cannot bypass the owner's pending PRD approval by
// calling save_file instead of delegate_agent.
func (s *Service) AuthorizeExternalFileWrite(ctx context.Context, userID, projectID, taskID string) error {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.ProjectID != projectID || task.Status == domain.StatusDone ||
		task.Status == domain.StatusFailed || task.Status == domain.StatusNeedsInput {
		return domain.ErrWrongState
	}
	if task.PRD != "" {
		state, err := s.ExternalApproval(ctx, userID, taskID)
		if err != nil {
			return err
		}
		if state.Status != "approved" {
			return domain.ErrWrongState
		}
	}
	return nil
}
