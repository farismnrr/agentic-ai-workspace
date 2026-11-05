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

func (s *Store) SubmitExternalLeadApproval(
	ctx context.Context, userID, taskID, source string, expectedRevision int64, event domain.Event, at time.Time,
) (domain.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Event{}, mapBusy(fmt.Errorf("begin external approval submission: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	// Claim only the exact PRD/plan revision inspected by the caller.
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'needs-input', updated_at = ?
		WHERE id = ? AND user_id = ? AND source = ? AND archived_at IS NULL
		  AND COALESCE(prd, '') <> '' AND COALESCE(json_array_length(plan), 0) > 0
		  AND ? = (SELECT COALESCE(MAX(seq), 0) FROM task_events
		      WHERE task_id = tasks.id AND
		        (kind = 'prd' OR (kind = 'plan' AND
		          COALESCE(json_extract(body, '$.text'), '') !=
		            'External orchestrator updated step progress')))
		  AND NOT EXISTS (
		       SELECT 1 FROM task_events
			       WHERE task_id = tasks.id AND kind = 'question'
				         AND COALESCE(json_extract(body, '$.agent'), '') = ?
				         AND COALESCE(json_extract(body, '$.answered'), '') = ''
				         AND seq > ?
		  ) AND status IN ('queued', 'planning', 'working', 'needs-input')`, at.Unix(), taskID, userID, source, expectedRevision, source, expectedRevision)
	if err != nil {
		return domain.Event{}, mapBusy(fmt.Errorf("set external approval state: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return domain.Event{}, mapBusy(fmt.Errorf("count external approval state changes: %w", err))
	}
	if changed != 1 {
		return domain.Event{}, domain.ErrWrongState
	}
	event.Time = at
	event, err = insertEvent(ctx, tx, s.ids, taskID, event)
	if err != nil {
		return domain.Event{}, mapBusy(fmt.Errorf("record external approval: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return domain.Event{}, mapBusy(fmt.Errorf("commit external approval: %w", err))
	}
	return event, nil
}

func (s *Store) AnswerExternalLeadQuestion(
	ctx context.Context, userID, taskID, source string, event domain.Event, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin external answer: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	var status, currentSource string
	err = tx.QueryRowContext(ctx, `SELECT status, source FROM tasks
		WHERE id = ? AND user_id = ? AND archived_at IS NULL`, taskID, userID).Scan(&status, &currentSource)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return mapBusy(fmt.Errorf("read external answer task: %w", err))
	}
	if currentSource != source || status != string(domain.StatusNeedsInput) ||
		event.Kind != domain.EventQuestion || event.Agent != source || event.Answered == "" {
		return domain.ErrNotWaiting
	}
	current, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+`
		FROM task_events WHERE task_id = ? AND id = ?`, taskID, event.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return mapBusy(fmt.Errorf("read pending external question: %w", err))
	}
	if current.Kind != domain.EventQuestion || current.Agent != source ||
		current.Answered != "" || current.Version != event.Version {
		return domain.ErrNotWaiting
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode external answer: %w", err)
	}
	updated, err := tx.ExecContext(ctx, `UPDATE task_events SET body = ?, version =
		(SELECT COALESCE(MAX(version), 0) + 1 FROM task_events WHERE task_id = ?)
		WHERE task_id = ? AND id = ? AND version = ?
		  AND COALESCE(json_extract(body, '$.answered'), '') = ''`,
		string(body), taskID, taskID, event.ID, event.Version)
	if err != nil {
		return mapBusy(fmt.Errorf("save external answer: %w", err))
	}
	if changed, err := updated.RowsAffected(); err != nil || changed != 1 {
		return domain.ErrNotWaiting
	}
	resumed, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'working', updated_at = ?
		WHERE id = ? AND user_id = ? AND source = ? AND status = 'needs-input' AND archived_at IS NULL`,
		at.Unix(), taskID, userID, source)
	if err != nil {
		return mapBusy(fmt.Errorf("resume external task after answer: %w", err))
	}
	if changed, err := resumed.RowsAffected(); err != nil || changed != 1 {
		return domain.ErrWrongState
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit external answer: %w", err))
	}
	return nil
}
