package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Store) AnswerExternalQuestion(ctx context.Context, userID, projectID, taskID, expectedWorkID, eventID string,
	answers [][]string, summary string,
) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, mapBusy(err)
	}
	defer func() { _ = tx.Rollback() }()
	if lockErr := lockAnswerWork(ctx, tx, userID, projectID, taskID); lockErr != nil {
		return false, lockErr
	}
	workID, err := openQuestionWork(ctx, tx, userID, projectID, taskID, expectedWorkID, eventID)
	if err != nil {
		return false, err
	}
	saveErr := saveQuestionAnswer(ctx, tx, userID, projectID, taskID, eventID, workID, answers, summary)
	if saveErr != nil {
		return false, saveErr
	}
	last, err := resumeWhenQuestionsClear(ctx, tx, workID, taskID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, mapBusy(err)
	}
	return last, nil
}

func lockAnswerWork(ctx context.Context, tx *sql.Tx, userID, projectID, taskID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE external_works SET updated_at = updated_at
		WHERE user_id = ? AND project_id = ? AND task_id = ?`, userID, projectID, taskID)
	return mapBusy(err)
}

func openQuestionWork(
	ctx context.Context, tx *sql.Tx, userID, projectID, taskID, workID, eventID string,
) (string, error) {
	var foundWork string
	err := tx.QueryRowContext(ctx, `SELECT work_id FROM external_work_questions WHERE user_id = ? AND project_id = ?
		AND task_id = ? AND work_id = ? AND event_id = ? AND state = 'open'`,
		userID, projectID, taskID, workID, eventID).Scan(&foundWork)
	if err != nil {
		return "", domain.ErrNotWaiting
	}
	return foundWork, nil
}

func saveQuestionAnswer(ctx context.Context, tx *sql.Tx, userID, projectID, taskID, eventID, workID string,
	answers [][]string, summary string,
) error {
	encoded, err := json.Marshal(answers)
	if err != nil {
		return fmt.Errorf("encode work question answers: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE external_work_questions SET state = 'answered', answers = ?,
		summary = ?, answered_at = unixepoch(), answered_by = ?, answer_source = 'app'
		WHERE work_id = ? AND user_id = ? AND project_id = ? AND task_id = ? AND event_id = ? AND state = 'open'`,
		string(encoded), summary, userID, workID, userID, projectID, taskID, eventID)
	if err != nil {
		return mapBusy(fmt.Errorf("save work question answer: %w", err))
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return domain.ErrNotWaiting
	}
	event, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+
		` FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if err != nil {
		return fmt.Errorf("read question card: %w", err)
	}
	event.Answered = summary
	return mapBusy(updateEventIn(ctx, tx, taskID, event))
}

func resumeWhenQuestionsClear(ctx context.Context, tx *sql.Tx, workID, taskID string) (bool, error) {
	var open int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_work_questions WHERE work_id = ? AND state = 'open'`,
		workID).Scan(&open); err != nil {
		return false, fmt.Errorf("count open work questions: %w", err)
	}
	if open != 0 {
		return false, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE external_works SET plan_state = 'draft', rev = rev + 1,
		updated_at = unixepoch() WHERE id = ? AND state = 'open' AND plan_state = 'awaiting_clarification'`, workID)
	if err != nil {
		return false, mapBusy(err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count resumed work update: %w", err)
	}
	if updated != 1 {
		return false, domain.ErrWrongState
	}
	result, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'working', updated_at = unixepoch()
		WHERE id = ? AND status = 'needs-input'`, taskID)
	if err != nil {
		return false, mapBusy(err)
	}
	updated, err = result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count resumed task update: %w", err)
	}
	if updated != 1 {
		return false, domain.ErrWrongState
	}
	return true, nil
}
