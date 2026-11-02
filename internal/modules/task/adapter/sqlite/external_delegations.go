package sqlite

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ListDelegations reads the existing same-task agent job records without
// introducing a second persistence model or a schema migration.
func (s *Store) ListDelegations(ctx context.Context, taskID string) ([]application.Delegation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, role_key, role_name, provider_id,
		model, effort, brief, status, report, start_seq
		FROM task_delegations WHERE task_id = ? ORDER BY started_at, id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list delegations: %w", err)
	}
	defer rows.Close()
	var out []application.Delegation
	for rows.Next() {
		var d application.Delegation
		var effort string
		if err := rows.Scan(&d.ID, &d.RoleKey, &d.RoleName, &d.ProviderID, &d.Model,
			&effort, &d.Brief, &d.Status, &d.Report, &d.StartSeq); err != nil {
			return nil, fmt.Errorf("read delegation: %w", err)
		}
		d.Effort = domain.Effort(effort)
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan delegations: %w", err)
	}
	return out, nil
}
