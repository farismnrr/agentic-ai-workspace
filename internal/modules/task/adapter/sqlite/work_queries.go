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

// FindOp returns a saved idempotent response when the operation exists.
func (s *Store) FindOp(ctx context.Context, workID, opKey string) (application.WorkOp, bool, error) {
	var op application.WorkOp
	err := s.db.QueryRowContext(ctx, `SELECT work_id, op_key, tool, req_hash, response, at
		FROM external_work_ops WHERE work_id = ? AND op_key = ?`, workID, opKey).
		Scan(&op.WorkID, &op.OpKey, &op.Tool, &op.ReqHash, &op.Response, &op.At)
	if errors.Is(err, sql.ErrNoRows) {
		return application.WorkOp{}, false, nil
	}
	if err != nil {
		return application.WorkOp{}, false, fmt.Errorf("find work operation: %w", err)
	}
	return op, true, nil
}

// StaleOpen returns open works inactive at or before the cutoff.
func (s *Store) StaleOpen(ctx context.Context, userID string, before time.Time) ([]domain.ExternalWork, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+workColumns+` FROM external_works
		WHERE user_id = ? AND state = 'open' AND updated_at <= ? ORDER BY updated_at`, userID, before.Unix())
	if err != nil {
		return nil, fmt.Errorf("select stale works: %w", err)
	}
	defer func() { _ = rows.Close() }()

	works := []domain.ExternalWork{}
	for rows.Next() {
		work, err := scanWork(rows)
		if err != nil {
			return nil, err
		}
		works = append(works, work)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stale works: %w", err)
	}
	return works, nil
}

// CountOpen counts a user's open works, optionally within one project.
func (s *Store) CountOpen(ctx context.Context, userID, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_works
		WHERE user_id = ? AND state = 'open' AND (? = '' OR project_id = ?)`,
		userID, projectID, projectID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count open works: %w", err)
	}
	return count, nil
}

func scanWork(row scanner) (domain.ExternalWork, error) {
	var work domain.ExternalWork
	var state string
	var created, updated int64
	var ended sql.NullInt64
	err := row.Scan(&work.ID, &work.TaskID, &work.UserID, &work.ProjectID, &work.Client, &state,
		&work.RoleKey, &work.RoleName, &work.Rev, &work.PlanRev, &work.NextStep, &work.IdemKey,
		&work.IdemHash, &created, &updated, &ended)
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("scan work: %w", err)
	}
	work.State = domain.WorkState(state)
	work.CreatedAt, work.UpdatedAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC()
	if ended.Valid {
		work.EndedAt = ended.Int64
	}
	return work, nil
}
