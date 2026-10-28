package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// AppendWorkCall stores an event with the work's active role while the row is locked for writing.
func (s *Store) AppendWorkCall(
	ctx context.Context, userID, client, projectID, workID string, event domain.Event, reported bool,
) (string, domain.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", domain.Event{}, fmt.Errorf("begin work call: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	taskID, stored, replay, err := appendWorkCall(ctx, tx, s, userID, client, projectID, workID, event, reported)
	if err != nil {
		return "", domain.Event{}, err
	}
	if replay {
		return taskID, stored, nil
	}
	if err := tx.Commit(); err != nil {
		return "", domain.Event{}, fmt.Errorf("commit work call: %w", err)
	}
	return taskID, stored, nil
}

func appendWorkCall(ctx context.Context, tx *sql.Tx, s *Store,
	userID, client, projectID, workID string, event domain.Event, reported bool,
) (string, domain.Event, bool, error) {
	var activeTaskID string
	lookupErr := tx.QueryRowContext(ctx, `SELECT task_id FROM external_works
		WHERE id = ? AND user_id = ? AND client = ? AND project_id = ?
		AND state IN ('open','blocked')`, workID, userID, client, projectID).Scan(&activeTaskID)
	if lookupErr == nil {
		prior, found, findErr := existingWorkCall(ctx, tx, activeTaskID, workID, event.ToolUseID)
		if findErr != nil {
			return "", domain.Event{}, false, findErr
		}
		if found {
			return activeTaskID, prior, true, nil
		}
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		return "", domain.Event{}, false, fmt.Errorf("find active work call: %w", lookupErr)
	}
	taskID, role, roleName, err := touchWorkCall(ctx, tx, userID, client, projectID, workID, s.now())
	if err != nil {
		return "", domain.Event{}, false, err
	}
	stored, found, err := storeWorkCall(ctx, tx, s.ids, taskID, client, role, roleName, workID, event, reported)
	if err != nil {
		return "", domain.Event{}, false, err
	}
	if found {
		return taskID, stored, true, nil
	}
	if _, updateErr := tx.ExecContext(ctx,
		`UPDATE tasks SET updated_at = ? WHERE id = ?`, stored.Time.Unix(), taskID); updateErr != nil {
		return "", domain.Event{}, false, fmt.Errorf("update work task activity time: %w", updateErr)
	}
	return taskID, stored, false, nil
}

func existingWorkCall(ctx context.Context, tx *sql.Tx, taskID, workID, callID string) (domain.Event, bool, error) {
	if callID == "" {
		return domain.Event{}, false, nil
	}
	event, err := scanEvent(tx.QueryRowContext(ctx,
		`SELECT `+eventColumns+` FROM task_events WHERE task_id = ? AND tool_use_id = ?`, taskID, callID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, false, nil
	}
	if err != nil {
		return domain.Event{}, false, fmt.Errorf("find connected-app call event: %w", err)
	}
	if event.Work == nil || event.Work.ID != workID {
		return domain.Event{}, false, domain.ErrBadRequest
	}
	return event, true, nil
}

func touchWorkCall(
	ctx context.Context, tx *sql.Tx, userID, client, projectID, workID string,
	now time.Time,
) (string, string, string, error) {
	var taskID, role, roleName string
	err := tx.QueryRowContext(ctx, `UPDATE external_works SET rev = rev + 1, updated_at = ?, reported_at = NULL
		WHERE id = ? AND user_id = ? AND client = ? AND project_id = ? AND state IN ('open','blocked')
		RETURNING task_id, role_key, role_name`, now.Unix(), workID, userID, client, projectID).
		Scan(&taskID, &role, &roleName)
	if err == nil {
		return taskID, role, roleName, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", "", fmt.Errorf("touch work call: %w", err)
	}
	var state string
	err = tx.QueryRowContext(ctx, `SELECT state FROM external_works
		WHERE id = ? AND user_id = ? AND client = ? AND project_id = ?`,
		workID, userID, client, projectID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", domain.ErrUnknownWork
	}
	if err != nil {
		return "", "", "", fmt.Errorf("check work call state: %w", err)
	}
	return "", "", "", domain.ErrWorkClosed
}

func touchWork(ctx context.Context, tx *sql.Tx, now time.Time, workID, taskID string) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE external_works SET rev = rev + 1, updated_at = ?, reported_at = NULL WHERE id = ? AND task_id = ?`,
		now.Unix(), workID, taskID); err != nil {
		return fmt.Errorf("update work activity time: %w", err)
	}
	return nil
}
