package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// externalPlanApprover is deliberately separate from the shared EventStore port:
// only External Orchestrator tasks can use this mutation.
type externalPlanApprover interface {
	ApproveExternalPlanEvent(ctx context.Context, userID, client, taskID, eventID, requestHash string, at time.Time) error
}

// ApproveExternalPlan approves exactly the pending owner question for the
// currently submitted PRD/plan revision. MCP scope and the owner's separate
// disabled-by-default tool switch are enforced by the Orchestrator adapter.
func (s *Service) ApproveExternalPlan(ctx context.Context, userID, client, taskID, eventID, key string) (ExternalPlanApproval, error) {
	client, taskID, eventID, key = strings.TrimSpace(client), strings.TrimSpace(taskID), strings.TrimSpace(eventID), strings.TrimSpace(key)
	if client == "" || taskID == "" || eventID == "" || len(key) < 8 || len(key) > 128 {
		return ExternalPlanApproval{}, domain.ErrBadRequest
	}
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	// Never approve another connected app's task, even for the same owner.
	if task.Source != externalLeadSource+client {
		return ExternalPlanApproval{}, domain.ErrNotFound
	}
	writer, ok := s.deps.Events.(externalPlanApprover)
	if !ok {
		return ExternalPlanApproval{}, domain.ErrWrongState
	}
	sum := sha256.Sum256([]byte(userID + "\x00" + client + "\x00" + taskID + "\x00" + eventID + "\x00" + key))
	if err := writer.ApproveExternalPlanEvent(ctx, userID, client, taskID, eventID, hex.EncodeToString(sum[:]), s.now()); err != nil {
		return ExternalPlanApproval{}, err
	}
	return ExternalPlanApproval{Status: "approved", EventID: eventID}, nil
}
