package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func storeWorkCall(
	ctx context.Context, tx *sql.Tx, ids application.IDs, taskID, client, role, roleName, workID string,
	event domain.Event, reported bool,
) (domain.Event, bool, error) {
	prior, found, err := existingWorkCall(ctx, tx, taskID, workID, event.ToolUseID)
	if err != nil || found {
		return prior, found, err
	}
	if err = checkWorkEventLimit(ctx, tx, taskID, 1); err != nil {
		return domain.Event{}, false, err
	}
	if event.Work == nil {
		event.Work = &domain.WorkRef{}
	}
	event.Work.ID, event.Work.Actor, event.Work.Role, event.Work.RoleName = workID, client, role, roleName
	stored, err := insertEvent(ctx, tx, ids, taskID, event)
	if err != nil {
		return domain.Event{}, false, err
	}
	if reported {
		if _, err := tx.ExecContext(ctx, `UPDATE external_works SET reported_at = ? WHERE id = ?`,
			stored.Time.Unix(), workID); err != nil {
			return domain.Event{}, false, fmt.Errorf("record reported work: %w", err)
		}
	}
	return stored, false, nil
}
