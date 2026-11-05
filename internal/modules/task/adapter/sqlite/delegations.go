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

// Scope returns the transcript of one delegation, kept apart from the orchestrator's own.
func (s *Store) Scope(scope string) application.TranscriptStore {
	return scopedTranscript{store: s, scope: scope}
}

type scopedTranscript struct {
	store *Store
	scope string
}

func (t scopedTranscript) AppendBlock(ctx context.Context, taskID string, role domain.Role, block domain.Block) error {
	return t.store.appendIn(ctx, taskID, t.scope, role, block)
}

func (t scopedTranscript) LoadTranscript(ctx context.Context, taskID string) ([]domain.Message, error) {
	return t.store.loadIn(ctx, taskID, t.scope)
}

// StartDelegation records a job handed to a teammate.
func (s *Store) StartDelegation(ctx context.Context, taskID string, d application.Delegation, at time.Time) error {
	scope, err := json.Marshal(d.Scope)
	if err != nil {
		return fmt.Errorf("encode delegation scope: %w", err)
	}
	mutating := 0
	if d.Mutating {
		mutating = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin delegation start: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO task_delegations (task_id, id, role_key, role_name, provider_id, model, effort, brief, status,
		   start_seq, started_at, execution_context, request_effort, execution_scope, mutating)
		 SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		 WHERE EXISTS (SELECT 1 FROM tasks WHERE id = ? AND status IN ('queued', 'planning', 'working'))
		   AND (SELECT COUNT(*) FROM task_delegations
		     WHERE task_id = ? AND status IN ('working', 'waiting')) < 4
		   AND (? = 0 OR NOT EXISTS (SELECT 1 FROM task_delegations
		     WHERE task_id = ? AND mutating = 1 AND status IN ('working', 'waiting')))`,
		taskID, d.ID, d.RoleKey, d.RoleName, d.ProviderID, d.Model, string(d.Effort), d.Brief, d.Status,
		d.StartSeq, at.Unix(), d.Context, d.RequestEffort, string(scope), mutating, taskID,
		taskID, mutating, taskID)
	if err != nil {
		return mapBusy(fmt.Errorf("insert delegation: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count delegation starts: %w", err)
	}
	if changed != 1 {
		return domain.ErrWrongState
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit delegation start: %w", err))
	}
	return nil
}

// FindDelegation loads one delegation; found is false when there is none.
func (s *Store) FindDelegation(ctx context.Context, taskID, id string) (application.Delegation, bool, error) {
	var (
		d      application.Delegation
		effort string
		scope  string
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, role_key, role_name, provider_id, model, effort, brief, status, report, start_seq,
		        execution_context, request_effort, execution_scope, mutating
		 FROM task_delegations WHERE task_id = ? AND id = ?`, taskID, id).
		Scan(&d.ID, &d.RoleKey, &d.RoleName, &d.ProviderID, &d.Model, &effort, &d.Brief, &d.Status, &d.Report,
			&d.StartSeq, &d.Context, &d.RequestEffort, &scope, &d.Mutating)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Delegation{}, false, nil
	}

	if err != nil {
		return application.Delegation{}, false, fmt.Errorf("select delegation: %w", err)
	}

	d.Effort = domain.Effort(effort)
	if err := json.Unmarshal([]byte(scope), &d.Scope); err != nil {
		return application.Delegation{}, false, fmt.Errorf("decode delegation scope: %w", err)
	}

	return d, true, nil
}

// SetDelegationStatus moves an open delegation between working, waiting and cancelled.
func (s *Store) SetDelegationStatus(ctx context.Context, taskID, id, status string) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE task_delegations SET status = ? WHERE task_id = ? AND id = ?
		 AND status IN ('working', 'waiting')`, status, taskID, id)
	if err != nil {
		return fmt.Errorf("update delegation: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count delegation status updates: %w", err)
	}
	if changed != 1 {
		var current string
		err := s.db.QueryRowContext(ctx, `SELECT status FROM task_delegations WHERE task_id = ? AND id = ?`,
			taskID, id).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrWrongState
		}
		if err != nil {
			return fmt.Errorf("read delegation status: %w", err)
		}
		if current != status {
			return domain.ErrWrongState
		}
	}
	return nil
}

// EndDelegation closes a delegation with its final status and report.
func (s *Store) EndDelegation(ctx context.Context, taskID, id, status, report string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE task_delegations SET status = ?, report = ?, ended_at = ? WHERE task_id = ? AND id = ?
		 AND status IN ('working', 'waiting')`,
		status, report, at.Unix(), taskID, id); err != nil {
		return fmt.Errorf("end delegation: %w", err)
	}

	return nil
}
