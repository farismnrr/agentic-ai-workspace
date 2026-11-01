package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// External Security is an opt-in review role. Neither the connected assistant
// nor its reported role may elevate the owner's persisted task mode.
func externalRoleModeAllowed(mode domain.ReviewMode, role string) error {
	if canonicalMemberKey(role) == "security" && mode != domain.ReviewModeDeepReview {
		return domain.ErrRoleForbidden
	}
	return nil
}

func (w *Works) externalWorkRolesAllowed(
	ctx context.Context, userID string, work domain.ExternalWork, requested string, planned []string,
) error {
	task, err := w.deps.Tasks.Find(ctx, userID, work.TaskID)
	if err != nil || task.ProjectID != work.ProjectID || task.WorkID != work.ID ||
		task.Source != work.Client || task.ArchivedAt != nil {
		return domain.ErrUnknownWork
	}
	if err := externalRoleModeAllowed(task.ReviewMode, work.RoleKey); err != nil {
		return err
	}
	if err := externalRoleModeAllowed(task.ReviewMode, requested); err != nil {
		return err
	}
	for _, role := range planned {
		if err := externalRoleModeAllowed(task.ReviewMode, role); err != nil {
			return err
		}
	}
	return nil
}
