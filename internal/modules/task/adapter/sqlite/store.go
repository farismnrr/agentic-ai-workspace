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

// legacyMode fills the tasks.mode column, which predates the single approval policy.
// The journal is append-only, so the column stays; nothing reads it.
const legacyMode = "risky"

const taskColumns = `id, project_id, user_id, title, summary, status, provider_id, model, plan, proof, ` +
	`created_at, updated_at, source, context_used, context_window, effort, team, prd, research, ` +
	`plan_first, follow_settings, pinned_at, archived_at`

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
		   created_at, updated_at, source, effort, team, plan_first, follow_settings)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.UserID, t.Title, string(t.Status), legacyMode, t.ProviderID, t.Model, string(plan),
		t.CreatedAt.Unix(), t.UpdatedAt.Unix(), t.Source, string(t.Effort), t.Team, t.PlanFirst, t.FollowSettings)
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

// Pin keeps the original pin time on repeated calls and hides tasks the person does not own.
func (s *Store) Pin(ctx context.Context, userID, id string, at time.Time) (domain.Task, error) {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET pinned_at = COALESCE(pinned_at, ?) WHERE id = ? AND user_id = ?`,
		at.Unix(), id, userID); err != nil {
		return domain.Task{}, fmt.Errorf("pin task: %w", err)
	}

	return s.Find(ctx, userID, id)
}

// Unpin clears a pin and hides tasks the person does not own.
func (s *Store) Unpin(ctx context.Context, userID, id string) (domain.Task, error) {
	const query = `UPDATE tasks SET pinned_at = NULL WHERE id = ? AND user_id = ?`
	if _, err := s.db.ExecContext(ctx, query, id, userID); err != nil {
		return domain.Task{}, fmt.Errorf("unpin task: %w", err)
	}

	return s.Find(ctx, userID, id)
}

// List returns a person's tasks, newest activity first.
func (s *Store) List(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error) {
	return s.list(ctx, userID, projectID, limit, `archived_at IS NULL`)
}

// All returns active and archived tasks for server maintenance and document exports.
func (s *Store) All(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error) {
	return s.list(ctx, userID, projectID, limit, "")
}

func (s *Store) list(ctx context.Context, userID, projectID string, limit int, state string) ([]domain.Task, error) {
	query, args := `SELECT `+taskColumns+` FROM tasks WHERE user_id = ?`, []any{userID}
	if projectID != "" {
		query, args = query+` AND project_id = ?`, append(args, projectID)
	}
	if state != "" {
		// #nosec G202 -- state is a fixed internal predicate, never request input.
		query += ` AND ` + state
	}

	query += ` ORDER BY updated_at DESC, rowid DESC`
	if limit >= 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}

	return collectTasks(rows)
}

// Running lists tasks left queued, planning or working.
func (s *Store) Running(ctx context.Context) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status IN ('queued', 'planning', 'working') AND archived_at IS NULL`)
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

// SavePRD records the product requirements document the team builds from.
func (s *Store) SavePRD(ctx context.Context, id, prd string, at time.Time) error {
	return s.exec(ctx, `UPDATE tasks SET prd = ?, updated_at = ? WHERE id = ?`, prd, at.Unix(), id)
}

// SaveResearch records the research notes the team builds from.
func (s *Store) SaveResearch(ctx context.Context, id, research string, at time.Time) error {
	return s.exec(ctx, `UPDATE tasks SET research = ?, updated_at = ? WHERE id = ?`, research, at.Unix(), id)
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
