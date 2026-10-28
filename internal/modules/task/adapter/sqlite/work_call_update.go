package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
	event, work, err := updateWorkCallEvent(ctx, tx, taskID, eventID, prior, event)
	if err != nil {
		return domain.Event{}, err
	}
	if err := updateWorkCallActivity(ctx, tx, s.now(), workID, taskID, work.Phase); err != nil {
		return domain.Event{}, err
	}
	if err := touchTask(ctx, tx, s.now(), taskID); err != nil {
		return domain.Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Event{}, fmt.Errorf("commit work call update: %w", err)
	}
	return event, nil
}

func updateWorkCallActivity(
	ctx context.Context, tx *sql.Tx, now time.Time, workID, taskID, phase string,
) error {
	if phase != "interrupted" {
		return touchWork(ctx, tx, now, workID, taskID)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE external_works SET rev = rev + 1, reported_at = NULL WHERE id = ? AND task_id = ?`,
		workID, taskID); err != nil {
		return fmt.Errorf("clear reported work: %w", err)
	}
	return nil
}

func updateWorkCallEvent(
	ctx context.Context,
	tx *sql.Tx,
	taskID, eventID string,
	prior, event domain.Event,
) (domain.Event, domain.WorkRef, error) {
	work := *prior.Work
	if event.Work != nil {
		work.Evidence, work.Phase = event.Work.Evidence, event.Work.Phase
	}
	event.ID, event.Seq, event.Time, event.Work = prior.ID, prior.Seq, prior.Time, &work
	body, err := encodeEvent(event)
	if err != nil {
		return domain.Event{}, domain.WorkRef{}, err
	}
	if err := tx.QueryRowContext(ctx, `UPDATE task_events SET body = ?, undo = ?, version =
		(SELECT COALESCE(MAX(version), 0) + 1 FROM task_events WHERE task_id = ?)
		WHERE task_id = ? AND id = ? RETURNING version`,
		body, event.Undo, taskID, taskID, eventID).Scan(&event.Version); err != nil {
		return domain.Event{}, domain.WorkRef{}, fmt.Errorf("update work call event: %w", err)
	}
	return event, work, nil
}

func touchTask(ctx context.Context, tx *sql.Tx, now time.Time, taskID string) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET updated_at = ? WHERE id = ?`, now.Unix(), taskID); err != nil {
		return fmt.Errorf("update work task activity time: %w", err)
	}
	return nil
}

func encodeEvent(event domain.Event) (string, error) {
	encoded, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode work call event: %w", err)
	}
	return string(encoded), nil
}
