package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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

	return s.exec(ctx, `UPDATE tasks SET status = 'review', summary = ?, proof = ?, updated_at = ? WHERE id = ? AND status <> 'done'`,
		summary, string(encoded), at.Unix(), id)
}
