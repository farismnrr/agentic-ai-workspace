package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// FollowUp atomically records the owner message, sticky mode promotion and queue state.
func (s *Store) FollowUp(ctx context.Context, userID, taskID string, event domain.Event, prompt string,
	mode domain.ReviewMode, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin task follow-up: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	status, archived, source, savedMode, err := followUpState(ctx, tx, userID, taskID)
	if err != nil {
		return err
	}
	mode, err = followUpMode(status, archived, source, savedMode, mode)
	if err != nil {
		return err
	}
	if _, insertErr := insertEvent(ctx, tx, s.ids, taskID, event); insertErr != nil {
		return mapBusy(fmt.Errorf("record task follow-up: %w", insertErr))
	}
	if transcriptErr := appendTranscript(ctx, tx, taskID, domain.RoleUser,
		domain.Block{Kind: domain.BlockText, Text: prompt}, ""); transcriptErr != nil {
		return mapBusy(fmt.Errorf("record task message: %w", transcriptErr))
	}
	_, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'queued', review_mode = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`, string(mode), at.Unix(), taskID, userID)
	if err != nil {
		return mapBusy(fmt.Errorf("queue task follow-up: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit task follow-up: %w", err))
	}
	return nil
}

func followUpMode(
	status domain.Status, archived sql.NullInt64, source string, saved, requested domain.ReviewMode,
) (domain.ReviewMode, error) {
	if !requested.Valid() {
		return "", domain.ErrBadMode
	}
	if archived.Valid || source != "" {
		return "", domain.ErrWrongState
	}
	if status.Running() {
		return "", domain.ErrBusy
	}
	if status == domain.StatusNeedsInput {
		return "", domain.ErrNeedsAnswer
	}
	if saved == domain.ReviewModeDeepReview && requested == domain.ReviewModeSwift {
		return "", domain.ErrBadMode
	}
	if requested != domain.ReviewModeDeepReview {
		requested = saved
	}
	return requested, nil
}

func followUpState(ctx context.Context, tx *sql.Tx, userID, taskID string) (
	domain.Status, sql.NullInt64, string, domain.ReviewMode, error,
) {
	var status, source, mode string
	var archived sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT status, archived_at, source, review_mode
		FROM tasks WHERE id = ? AND user_id = ?`, taskID, userID).Scan(&status, &archived, &source, &mode)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.NullInt64{}, "", "", domain.ErrNotFound
	}
	if err != nil {
		return "", sql.NullInt64{}, "", "", mapBusy(fmt.Errorf("read task follow-up state: %w", err))
	}
	return domain.Status(status), archived, source, domain.ReviewMode(mode), nil
}
