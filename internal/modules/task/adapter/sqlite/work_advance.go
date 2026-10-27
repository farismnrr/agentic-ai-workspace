package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func advanceWork(ctx context.Context, tx *sql.Tx, change application.WorkChange) (int64, error) {
	var patch application.WorkPatch
	if change.Patch != nil {
		patch = *change.Patch
	}
	result, err := tx.ExecContext(ctx, `UPDATE external_works SET rev = rev + 1, updated_at = ?,
		state = COALESCE(?, state), role_key = COALESCE(?, role_key), role_name = COALESCE(?, role_name),
		plan_rev = COALESCE(?, plan_rev), next_step = COALESCE(?, next_step), ended_at = COALESCE(?, ended_at),
		plan_state = COALESCE(?, plan_state)
		WHERE id = ? AND user_id = ? AND rev = ?`, time.Now().Unix(), value(patch.State), value(patch.RoleKey),
		value(patch.RoleName), value(patch.PlanRev), value(patch.NextStep), nullTime(patch.EndedAt), value(patch.PlanState),
		change.WorkID, change.UserID, change.ExpectRev)
	if err != nil {
		return 0, fmt.Errorf("advance work revision: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count work revision: %w", err)
	}
	if changed == 0 {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM external_works WHERE id = ? AND user_id = ?`,
			change.WorkID, change.UserID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrUnknownWork
		}
		if err != nil {
			return 0, fmt.Errorf("check work after revision conflict: %w", err)
		}
		return 0, domain.ErrStaleRevision
	}
	return change.ExpectRev + 1, nil
}
