package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// AppendWorkCall stores an event with the work's active role while the row is locked for writing.
func (s *Store) AppendWorkCall(
	ctx context.Context, userID, client, projectID, workID string, event domain.Event,
) (string, domain.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", domain.Event{}, fmt.Errorf("begin work call: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	taskID, role, roleName, err := touchWorkCall(ctx, tx, userID, client, projectID, workID)
	if err != nil {
		return "", domain.Event{}, err
	}
	if event.Work == nil {
		event.Work = &domain.WorkRef{}
	}
	event.Work.ID, event.Work.Actor, event.Work.Role, event.Work.RoleName = workID, client, role, roleName
	stored, err := insertEvent(ctx, tx, s.ids, taskID, event)
	if err != nil {
		return "", domain.Event{}, err
	}
	if _, updateErr := tx.ExecContext(ctx,
		`UPDATE tasks SET updated_at = ? WHERE id = ?`, stored.Time.Unix(), taskID); updateErr != nil {
		return "", domain.Event{}, fmt.Errorf("update work task activity time: %w", updateErr)
	}
	if err := tx.Commit(); err != nil {
		return "", domain.Event{}, fmt.Errorf("commit work call: %w", err)
	}
	return taskID, stored, nil
}

func touchWorkCall(
	ctx context.Context, tx *sql.Tx, userID, client, projectID, workID string,
) (string, string, string, error) {
	var taskID, role, roleName string
	err := tx.QueryRowContext(ctx, `UPDATE external_works SET updated_at = ?
		WHERE id = ? AND user_id = ? AND client = ? AND project_id = ? AND state IN ('open','blocked')
		RETURNING task_id, role_key, role_name`, time.Now().Unix(), workID, userID, client, projectID).
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

// UpdateWorkCall updates the start event while preserving its original role attribution.
func (s *Store) UpdateWorkCall(
	ctx context.Context, taskID, workID, eventID string, event domain.Event,
) (domain.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Event{}, fmt.Errorf("begin work call update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	prior, err := scanEvent(tx.QueryRowContext(ctx,
		`SELECT `+eventColumns+` FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if err != nil || prior.Work == nil || prior.Work.ID != workID {
		return domain.Event{}, domain.ErrNotFound
	}
	work := *prior.Work
	if event.Work != nil {
		work.Evidence, work.Phase = event.Work.Evidence, event.Work.Phase
	}
	event.ID, event.Seq, event.Time, event.Work = prior.ID, prior.Seq, prior.Time, &work
	body, err := encodeEvent(event)
	if err != nil {
		return domain.Event{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE task_events SET body = ?, undo = ? WHERE task_id = ? AND id = ?`,
		body, event.Undo, taskID, eventID); err != nil {
		return domain.Event{}, fmt.Errorf("update work call event: %w", err)
	}
	if _, updateErr := tx.ExecContext(ctx,
		`UPDATE tasks SET updated_at = ? WHERE id = ?`, time.Now().Unix(), taskID); updateErr != nil {
		return domain.Event{}, fmt.Errorf("update work task activity time: %w", updateErr)
	}
	if err := tx.Commit(); err != nil {
		return domain.Event{}, fmt.Errorf("commit work call update: %w", err)
	}
	return event, nil
}

func encodeEvent(event domain.Event) (string, error) {
	encoded, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode work call event: %w", err)
	}
	return string(encoded), nil
}
