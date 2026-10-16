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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_delegations (task_id, id, role_key, role_name, provider_id, model, effort, brief, status,
		   start_seq, started_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, d.ID, d.RoleKey, d.RoleName, d.ProviderID, d.Model, string(d.Effort), d.Brief, d.Status, d.StartSeq,
		at.Unix())
	if err != nil {
		return fmt.Errorf("insert delegation: %w", err)
	}

	return nil
}

// FindDelegation loads one delegation; found is false when there is none.
func (s *Store) FindDelegation(ctx context.Context, taskID, id string) (application.Delegation, bool, error) {
	var (
		d      application.Delegation
		effort string
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, role_key, role_name, provider_id, model, effort, brief, status, report, start_seq
		 FROM task_delegations WHERE task_id = ? AND id = ?`, taskID, id).
		Scan(&d.ID, &d.RoleKey, &d.RoleName, &d.ProviderID, &d.Model, &effort, &d.Brief, &d.Status, &d.Report,
			&d.StartSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Delegation{}, false, nil
	}

	if err != nil {
		return application.Delegation{}, false, fmt.Errorf("select delegation: %w", err)
	}

	d.Effort = domain.Effort(effort)

	return d, true, nil
}

// SetDelegationStatus moves a delegation between working and waiting.
func (s *Store) SetDelegationStatus(ctx context.Context, taskID, id, status string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE task_delegations SET status = ? WHERE task_id = ? AND id = ?`, status, taskID, id); err != nil {
		return fmt.Errorf("update delegation: %w", err)
	}

	return nil
}

// EndDelegation closes a delegation with its final status and report.
func (s *Store) EndDelegation(ctx context.Context, taskID, id, status, report string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE task_delegations SET status = ?, report = ?, ended_at = ? WHERE task_id = ? AND id = ?`,
		status, report, at.Unix(), taskID, id); err != nil {
		return fmt.Errorf("end delegation: %w", err)
	}

	return nil
}
