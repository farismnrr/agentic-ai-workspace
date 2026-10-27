package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Store) AskQuestion(ctx context.Context, in application.QuestionCreate) (
	application.WorkQuestion, bool, error,
) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.WorkQuestion{}, false, mapBusy(err)
	}
	defer func() { _ = tx.Rollback() }()
	if lockErr := lockQuestionWork(ctx, tx, in); lockErr != nil {
		return application.WorkQuestion{}, false, lockErr
	}
	prior, found, err := persistedQuestion(ctx, tx, in)
	if err != nil || found {
		if err != nil {
			return application.WorkQuestion{}, false, fmt.Errorf("check question retry: %w", err)
		}
		return prior.Question, found, nil
	}
	seq, err := nextQuestionSeq(ctx, tx, in.WorkID)
	if err != nil {
		return application.WorkQuestion{}, false, err
	}
	if err := storeQuestion(ctx, tx, s.ids, &in, seq); err != nil {
		return application.WorkQuestion{}, false, err
	}
	if err := advanceQuestionWork(ctx, tx, in); err != nil {
		return application.WorkQuestion{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return application.WorkQuestion{}, false, mapBusy(err)
	}
	return in.Question, false, nil
}

func lockQuestionWork(ctx context.Context, tx *sql.Tx, in application.QuestionCreate) error {
	_, err := tx.ExecContext(ctx, `UPDATE external_works SET updated_at = updated_at
		WHERE id = ? AND user_id = ? AND project_id = ? AND task_id = ?`,
		in.WorkID, in.UserID, in.ProjectID, in.TaskID)
	if err != nil {
		return mapBusy(err)
	}
	return nil
}

func persistedQuestion(ctx context.Context, tx *sql.Tx, in application.QuestionCreate) (
	application.QuestionCreate, bool, error,
) {
	if in.RequestKey == "" {
		return application.QuestionCreate{}, false, nil
	}
	prior, found, err := questionByKey(ctx, tx, in.WorkID, in.RequestKey)
	if err != nil || !found {
		return application.QuestionCreate{}, found, err
	}
	form, _ := json.Marshal(in.Question.Form)
	priorForm, _ := json.Marshal(prior.Form)
	if string(form) != string(priorForm) {
		return application.QuestionCreate{}, false, domain.ErrKeyReused
	}
	return application.QuestionCreate{Question: prior}, true, nil
}

func nextQuestionSeq(ctx context.Context, tx *sql.Tx, workID string) (int64, error) {
	var total, open int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE state = 'open')
		FROM external_work_questions WHERE work_id = ?`, workID).Scan(&total, &open); err != nil {
		return 0, fmt.Errorf("count work questions: %w", err)
	}
	if open >= application.MaxOpenWorkQuestions || total >= application.MaxWorkQuestions {
		return 0, domain.ErrTooMany
	}
	var seq int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM external_work_questions WHERE work_id = ?`,
		workID).Scan(&seq); err != nil {
		return 0, fmt.Errorf("number work question: %w", err)
	}
	return seq, nil
}

func storeQuestion(ctx context.Context, tx *sql.Tx, ids application.IDs,
	in *application.QuestionCreate, seq int64,
) error {
	form, err := json.Marshal(in.Question.Form)
	if err != nil {
		return fmt.Errorf("encode question form: %w", err)
	}
	event, err := insertEvent(ctx, tx, ids, in.TaskID, in.Event)
	if err != nil {
		return mapBusy(err)
	}
	in.Question.EventID = event.ID
	_, err = tx.ExecContext(ctx, `INSERT INTO external_work_questions
		(id, work_id, task_id, user_id, project_id, seq, request_key, form, state, event_id, asked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'open', ?, ?)`, in.Question.ID, in.WorkID, in.TaskID, in.UserID,
		in.ProjectID, seq, in.RequestKey, string(form), event.ID, in.Question.AskedAt)
	if err != nil {
		return mapBusy(fmt.Errorf("save work question: %w", err))
	}
	if err := supersedeSubmittedPlan(ctx, tx, in.WorkID); err != nil {
		return mapBusy(err)
	}
	return nil
}

func advanceQuestionWork(ctx context.Context, tx *sql.Tx, in application.QuestionCreate) error {
	if _, err := tx.ExecContext(ctx, `UPDATE external_works SET plan_state = 'awaiting_clarification',
		rev = rev + 1, updated_at = unixepoch() WHERE id = ? AND user_id = ?`, in.WorkID, in.UserID); err != nil {
		return mapBusy(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'needs-input', updated_at = unixepoch() WHERE id = ?`,
		in.TaskID); err != nil {
		return mapBusy(err)
	}
	return nil
}

var _ application.QuestionStore = (*Store)(nil)
