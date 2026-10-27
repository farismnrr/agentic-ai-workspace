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

func (s *Store) CreateHandoffTask(ctx context.Context, handoff application.PlanHandoff, task domain.Task,
	event domain.Event, prompt string, at time.Time,
) (domain.Task, bool, error) {
	if linked, err := s.linkedTask(ctx, handoff.TaskID, handoff.UserID, handoff.ProjectID); err == nil {
		return linked, false, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Task{}, false, mapBusy(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, false, mapBusy(fmt.Errorf("begin plan handoff: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	if insertErr := insertTask(ctx, tx, task); insertErr != nil {
		return s.handoffCreateRace(ctx, tx, handoff, insertErr)
	}
	attached, err := attachLinkedTask(ctx, tx, handoff.TaskID, task)
	if err != nil {
		return domain.Task{}, false, mapBusy(err)
	}
	if !attached {
		return s.handoffCreateRace(ctx, tx, handoff, domain.ErrNotFound)
	}
	if err := persistLinkedConversation(ctx, tx, s.ids, task, event, prompt); err != nil {
		return domain.Task{}, false, mapBusy(err)
	}
	if err := markHandoffCreated(ctx, tx, handoff, task.ID, at); err != nil {
		return domain.Task{}, false, mapBusy(err)
	}
	if err := tx.Commit(); err != nil {
		if linked, ok := s.recoverHandoffAfterCommit(ctx, handoff, err); ok {
			return linked, false, nil
		}
		return domain.Task{}, false, mapBusy(fmt.Errorf("commit plan handoff task: %w", err))
	}
	return task, true, nil
}

func (s *Store) handoffCreateRace(ctx context.Context, tx *sql.Tx, handoff application.PlanHandoff,
	cause error,
) (domain.Task, bool, error) {
	if err := tx.Rollback(); err != nil {
		return domain.Task{}, false, fmt.Errorf("rollback duplicate plan handoff: %w", err)
	}
	linked, err := s.linkedTaskAfterRace(ctx, handoff.TaskID, handoff.UserID, handoff.ProjectID)
	if err == nil {
		return linked, false, nil
	}
	if errors.Is(cause, domain.ErrNotFound) && errors.Is(err, domain.ErrNotFound) {
		return domain.Task{}, false, domain.ErrWrongState
	}
	return domain.Task{}, false, mapBusy(cause)
}

func (s *Store) recoverHandoffAfterCommit(ctx context.Context, handoff application.PlanHandoff,
	commitErr error,
) (domain.Task, bool) {
	if !isSQLiteBusy(commitErr) {
		return domain.Task{}, false
	}
	var taskID string
	err := s.db.QueryRowContext(ctx, `SELECT internal_task_id FROM external_plan_handoffs
		WHERE id = ? AND state IN ('created', 'done') AND internal_task_id IS NOT NULL`, handoff.ID).Scan(&taskID)
	if err != nil {
		return domain.Task{}, false
	}
	task, err := s.Find(ctx, handoff.UserID, taskID)
	return task, err == nil
}
