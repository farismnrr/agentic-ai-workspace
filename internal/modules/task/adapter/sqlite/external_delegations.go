package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ListDelegations reads the durable same-task agent job records.
func (s *Store) ListDelegations(ctx context.Context, taskID string) ([]application.Delegation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, role_key, role_name, provider_id,
		model, effort, brief, status, report, start_seq, execution_context, request_effort, execution_scope, mutating, parameters_verified
		FROM task_delegations WHERE task_id = ? ORDER BY started_at, id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list delegations: %w", err)
	}
	defer rows.Close()
	var out []application.Delegation
	for rows.Next() {
		var d application.Delegation
		var effort, scope string
		if err := rows.Scan(
			&d.ID, &d.RoleKey, &d.RoleName, &d.ProviderID, &d.Model, &effort, &d.Brief,
			&d.Status, &d.Report, &d.StartSeq, &d.Context, &d.RequestEffort, &scope, &d.Mutating,
			&d.ParametersVerified,
		); err != nil {
			return nil, fmt.Errorf("read delegation: %w", err)
		}
		d.Effort = domain.Effort(effort)
		if err := json.Unmarshal([]byte(scope), &d.Scope); err != nil {
			return nil, fmt.Errorf("decode delegation scope: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan delegations: %w", err)
	}
	return out, nil
}
