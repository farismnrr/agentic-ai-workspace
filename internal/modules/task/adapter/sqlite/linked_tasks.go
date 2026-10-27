package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// CreateLinkedTask creates the internal task, its opening request and the external-task link together.
func (s *Store) CreateLinkedTask(
	ctx context.Context,
	externalID string,
	task domain.Task,
	event domain.Event,
	prompt string,
) (domain.Task, bool, error) {
	if linked, err := s.linkedTask(ctx, externalID, task.UserID, task.ProjectID); err == nil {
		return linked, false, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, false, fmt.Errorf("begin linked task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if insertErr := insertTask(ctx, tx, task); insertErr != nil {
		_ = tx.Rollback()
		if linked, linkErr := s.linkedTaskAfterRace(ctx, externalID, task.UserID, task.ProjectID); linkErr == nil {
			return linked, false, nil
		}
		return domain.Task{}, false, insertErr
	}

	attached, err := attachLinkedTask(ctx, tx, externalID, task)
	if err != nil {
		return domain.Task{}, false, err
	}
	if !attached {
		return s.reuseLinkedTaskAfterRace(ctx, tx, externalID, task)
	}

	if err := persistLinkedConversation(ctx, tx, s.ids, task, event, prompt); err != nil {
		return domain.Task{}, false, err
	}
	if err := tx.Commit(); err != nil {
		if linked, recovered := s.recoverLinkedTaskAfterCommit(ctx, externalID, task, err); recovered {
			return linked, false, nil
		}
		return domain.Task{}, false, fmt.Errorf("commit linked chat: %w", err)
	}
	return task, true, nil
}

func (s *Store) reuseLinkedTaskAfterRace(
	ctx context.Context,
	tx *sql.Tx,
	externalID string,
	task domain.Task,
) (domain.Task, bool, error) {
	if err := tx.Rollback(); err != nil {
		return domain.Task{}, false, fmt.Errorf("rollback duplicate linked task: %w", err)
	}
	linked, err := s.linkedTaskAfterRace(ctx, externalID, task.UserID, task.ProjectID)
	return linked, false, err
}

func (s *Store) recoverLinkedTaskAfterCommit(
	ctx context.Context,
	externalID string,
	task domain.Task,
	commitErr error,
) (domain.Task, bool) {
	if !isSQLiteBusy(commitErr) {
		return domain.Task{}, false
	}
	linked, err := s.linkedTaskAfterRace(ctx, externalID, task.UserID, task.ProjectID)
	return linked, err == nil
}

func (s *Store) linkedTaskAfterRace(ctx context.Context, externalID, userID, projectID string) (domain.Task, error) {
	const attempts = 20
	const retryDelay = 10 * time.Millisecond
	for attempt := 0; attempt < attempts; attempt++ {
		task, err := s.linkedTask(ctx, externalID, userID, projectID)
		if err == nil {
			return task, nil
		}
		if !errors.Is(err, domain.ErrNotFound) && !isSQLiteBusy(err) {
			return domain.Task{}, err
		}
		if attempt == attempts-1 {
			return domain.Task{}, err
		}
		select {
		case <-ctx.Done():
			return domain.Task{}, fmt.Errorf("retry linked task lookup: %w", ctx.Err())
		case <-time.After(retryDelay):
		}
	}
	return domain.Task{}, domain.ErrNotFound
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "SQLITE_BUSY") || strings.Contains(message, "database is locked")
}

func attachLinkedTask(ctx context.Context, tx *sql.Tx, externalID string, task domain.Task) (bool, error) {
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET linked_task_id = ?
		WHERE id = ? AND user_id = ? AND project_id = ?
		AND (linked_task_id IS NULL OR EXISTS (
			SELECT 1 FROM tasks old WHERE old.id = tasks.linked_task_id AND old.archived_at IS NOT NULL
		))
		AND archived_at IS NULL AND (source <> '' OR work_id <> '')`,
		task.ID, externalID, task.UserID, task.ProjectID)
	if err != nil {
		return false, fmt.Errorf("link internal chat: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count linked task update: %w", err)
	}
	return changed == 1, nil
}

func persistLinkedConversation(
	ctx context.Context,
	tx *sql.Tx,
	ids application.IDs,
	task domain.Task,
	event domain.Event,
	prompt string,
) error {
	if _, err := insertEvent(ctx, tx, ids, task.ID, event); err != nil {
		return err
	}
	if err := appendTranscript(ctx, tx, task.ID, domain.RoleUser,
		domain.Block{Kind: domain.BlockText, Text: prompt}, ""); err != nil {
		return err
	}
	return nil
}

func (s *Store) linkedTask(ctx context.Context, externalID, userID, projectID string) (domain.Task, error) {
	var linkedID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT linked_task_id FROM tasks
		WHERE id = ? AND user_id = ? AND project_id = ? AND archived_at IS NULL
		AND (source <> '' OR work_id <> '')`, externalID, userID, projectID).Scan(&linkedID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("find external task link: %w", err)
	}
	if !linkedID.Valid {
		return domain.Task{}, domain.ErrNotFound
	}
	task, err := scanTask(s.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks
		WHERE id = ? AND user_id = ? AND project_id = ? AND origin_task_id = ? AND archived_at IS NULL
		AND source = '' AND work_id = ''`,
		linkedID.String, userID, projectID, externalID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, domain.ErrNotFound
	}
	return task, err
}
