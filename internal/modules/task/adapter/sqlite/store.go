package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const taskColumns = `id, project_id, user_id, title, summary, status, mode, provider_id, model, plan, proof, ` +
	`created_at, updated_at`

// Store persists tasks, their thread, the model transcript and usage in SQLite.
type Store struct {
	db  *sql.DB
	ids application.IDs
}

// NewStore wraps an open database; ids names new events.
func NewStore(db *sql.DB, ids application.IDs) *Store {
	return &Store{db: db, ids: ids}
}

type scanner interface {
	Scan(dest ...any) error
}

// Create inserts a task.
func (s *Store) Create(ctx context.Context, t domain.Task) error {
	plan, err := json.Marshal(append([]domain.PlanStep{}, t.Plan...))
	if err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, project_id, user_id, title, summary, status, mode, provider_id, model, plan,
		   created_at, updated_at) VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.UserID, t.Title, string(t.Status), string(t.Mode), t.ProviderID, t.Model, string(plan),
		t.CreatedAt.Unix(), t.UpdatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// Find loads a task; an empty userID skips the ownership check.
func (s *Store) Find(ctx context.Context, userID, id string) (domain.Task, error) {
	query, args := `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, []any{id}
	if userID != "" {
		query, args = query+` AND user_id = ?`, append(args, userID)
	}

	task, err := scanTask(s.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, domain.ErrNotFound
	}

	return task, err
}

// List returns a person's tasks, newest activity first.
func (s *Store) List(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error) {
	query, args := `SELECT `+taskColumns+` FROM tasks WHERE user_id = ?`, []any{userID}
	if projectID != "" {
		query, args = query+` AND project_id = ?`, append(args, projectID)
	}

	rows, err := s.db.QueryContext(ctx, query+` ORDER BY updated_at DESC, rowid DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}

	return collectTasks(rows)
}

// Running lists tasks left queued, planning or working.
func (s *Store) Running(ctx context.Context) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status IN ('queued', 'planning', 'working')`)
	if err != nil {
		return nil, fmt.Errorf("select running tasks: %w", err)
	}

	return collectTasks(rows)
}

// SetStatus changes a task's status.
func (s *Store) SetStatus(ctx context.Context, id string, status domain.Status, at time.Time) error {
	return s.exec(ctx, `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`, string(status), at.Unix(), id)
}

// SavePlan replaces a task's plan.
func (s *Store) SavePlan(ctx context.Context, id string, plan []domain.PlanStep, at time.Time) error {
	encoded, err := json.Marshal(append([]domain.PlanStep{}, plan...))
	if err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}

	return s.exec(ctx, `UPDATE tasks SET plan = ?, updated_at = ? WHERE id = ?`, string(encoded), at.Unix(), id)
}

// Finish hands a task to review with its summary and proof.
func (s *Store) Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error {
	encoded, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("encode proof: %w", err)
	}

	return s.exec(ctx, `UPDATE tasks SET status = 'review', summary = ?, proof = ?, updated_at = ? WHERE id = ?`,
		summary, string(encoded), at.Unix(), id)
}

func (s *Store) exec(ctx context.Context, query string, args ...any) error {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if changed, err := result.RowsAffected(); err == nil && changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func collectTasks(rows *sql.Rows) ([]domain.Task, error) {
	defer func() { _ = rows.Close() }()

	tasks := []domain.Task{}

	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	return tasks, nil
}

func scanTask(row scanner) (domain.Task, error) {
	var (
		t                  domain.Task
		status, mode, plan string
		proof              sql.NullString
		created, updated   int64
	)

	err := row.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Title, &t.Summary, &status, &mode, &t.ProviderID, &t.Model,
		&plan, &proof, &created, &updated)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	t.Status, t.Mode = domain.Status(status), domain.Mode(mode)
	t.CreatedAt, t.UpdatedAt = time.Unix(created, 0), time.Unix(updated, 0)

	if err := json.Unmarshal([]byte(plan), &t.Plan); err != nil {
		return domain.Task{}, fmt.Errorf("decode plan: %w", err)
	}

	if proof.Valid && proof.String != "null" {
		t.Proof = &domain.Proof{}
		if err := json.Unmarshal([]byte(proof.String), t.Proof); err != nil {
			return domain.Task{}, fmt.Errorf("decode proof: %w", err)
		}
	}

	return t, nil
}
