package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Store) Questions(ctx context.Context, workID string) ([]application.WorkQuestion, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, form, state, answers, summary, event_id, asked_at
		FROM external_work_questions WHERE work_id = ? AND state IN ('open', 'answered') ORDER BY seq`, workID)
	if err != nil {
		return nil, fmt.Errorf("read work questions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	questions := []application.WorkQuestion{}
	for rows.Next() {
		question, scanErr := scanWorkQuestion(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read work questions: %w", err)
	}
	return questions, nil
}

func (s *Store) OpenQuestionCount(ctx context.Context, workID string) (int, error) {
	return s.questionCount(ctx, workID, "open")
}

func (s *Store) TotalQuestionCount(ctx context.Context, workID string) (int, error) {
	return s.questionCount(ctx, workID, "")
}

func (s *Store) questionCount(ctx context.Context, workID, state string) (int, error) {
	query, args := `SELECT COUNT(*) FROM external_work_questions WHERE work_id = ?`, []any{workID}
	if state != "" {
		query += ` AND state = ?`
		args = append(args, state)
	}
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count work questions: %w", err)
	}
	return count, nil
}

func scanWorkQuestion(row scanner) (application.WorkQuestion, error) {
	var question application.WorkQuestion
	var form, answers string
	err := row.Scan(&question.ID, &form, &question.State, &answers, &question.Summary,
		&question.EventID, &question.AskedAt)
	if err != nil {
		return application.WorkQuestion{}, fmt.Errorf("scan work question: %w", err)
	}
	if err := json.Unmarshal([]byte(form), &question.Form); err != nil {
		return application.WorkQuestion{}, fmt.Errorf("decode work question form: %w", err)
	}
	if err := json.Unmarshal([]byte(answers), &question.Answers); err != nil {
		return application.WorkQuestion{}, fmt.Errorf("decode work question answers: %w", err)
	}
	return question, nil
}

func questionByKey(ctx context.Context, tx *sql.Tx, workID, key string) (application.WorkQuestion, bool, error) {
	question, err := scanWorkQuestion(tx.QueryRowContext(ctx, `SELECT id, form, state, answers, summary, event_id,
		asked_at FROM external_work_questions WHERE work_id = ? AND request_key = ?`, workID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return application.WorkQuestion{}, false, nil
	}
	return question, err == nil, err
}

func (s *Store) FindQuestion(ctx context.Context, userID, projectID, taskID, workID, eventID string) (
	application.WorkQuestion, domain.Event, error,
) {
	question, err := scanWorkQuestion(s.db.QueryRowContext(ctx, `SELECT id, form, state, answers, summary,
		event_id, asked_at FROM external_work_questions WHERE user_id = ? AND project_id = ? AND task_id = ?
		AND work_id = ? AND event_id = ?`, userID, projectID, taskID, workID, eventID))
	if err != nil {
		return application.WorkQuestion{}, domain.Event{}, domain.ErrNotFound
	}
	event, err := scanEvent(s.db.QueryRowContext(ctx, `SELECT `+eventColumns+
		` FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if err != nil {
		return application.WorkQuestion{}, domain.Event{}, domain.ErrNotFound
	}
	return question, event, nil
}
