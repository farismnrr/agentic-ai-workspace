package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// TaskLive reports whether the task exists in that project and is still open.
func (s *Store) TaskLive(ctx context.Context, projectID, taskID string) (bool, error) {
	var n int

	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks t WHERE t.id = ? AND t.project_id = ? AND `+liveTask,
		taskID, projectID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check task: %w", err)
	}

	return n == 1, nil
}

// EndInactive ends, in one statement, every pending or approved grant whose task is not live. An approved grant
// becomes revoked and ends now; a pending request becomes denied. It cannot be undone: a task that comes back to
// life does not get its grants back.
func (s *Store) EndInactive(ctx context.Context, now time.Time) (int, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE egress_grants SET
		status = CASE status WHEN 'approved' THEN 'revoked' ELSE 'denied' END,
		expires_at = CASE status WHEN 'approved' THEN ? ELSE expires_at END,
		decided_by = 'system:task-ended', decided_at = ?
		WHERE status IN ('pending', 'approved') AND NOT EXISTS (
			SELECT 1 FROM tasks t WHERE t.id = egress_grants.task_id AND t.project_id = egress_grants.project_id
			AND `+liveTask+`)`, now.Unix(), now.Unix())
	if err != nil {
		return 0, fmt.Errorf("end grants of finished tasks: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("end grants of finished tasks: %w", err)
	}

	return int(n), nil
}

// LiveApproved lists every approved grant that has not ended and whose task is live.
func (s *Store) LiveApproved(ctx context.Context, now time.Time) ([]domain.Grant, error) {
	return s.list(ctx, `SELECT `+prefixed("g")+` FROM egress_grants g
		JOIN tasks t ON t.id = g.task_id AND t.project_id = g.project_id AND `+liveTask+`
		WHERE g.status = 'approved' AND g.expires_at > ? ORDER BY g.created_at, g.id`, now.Unix())
}

// Live returns the pending or approved grant of the task for the host that has not ended.
func (s *Store) Live(ctx context.Context, taskID, host string, now time.Time) (domain.Grant, error) {
	return scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM egress_grants WHERE task_id = ? AND host = ?
		AND status IN ('pending', 'approved') AND expires_at > ? ORDER BY created_at DESC LIMIT 1`,
		taskID, host, now.Unix()))
}

// EndTask ends every pending or approved grant of the task, whatever state the task is in.
func (s *Store) EndTask(ctx context.Context, taskID string, now time.Time) (int, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE egress_grants SET
		status = CASE status WHEN 'approved' THEN 'revoked' ELSE 'denied' END,
		expires_at = CASE status WHEN 'approved' THEN ? ELSE expires_at END,
		decided_by = 'system:task-ended', decided_at = ?
		WHERE task_id = ? AND status IN ('pending', 'approved')`, now.Unix(), now.Unix(), taskID)
	if err != nil {
		return 0, fmt.Errorf("end grants of task: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("end grants of task: %w", err)
	}

	return int(n), nil
}
