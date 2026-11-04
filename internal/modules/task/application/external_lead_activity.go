package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// RecordExternalLeadAction records one server-observed MCP action on the already
// existing external-led task. The caller must supply a stable operation key.
// No connected app may use this method to attach an event to another app's task.
func (s *Service) RecordExternalLeadAction(
	ctx context.Context, userID, client, taskID, tool, operationKey, description string,
) error {
	if strings.TrimSpace(client) == "" || strings.TrimSpace(taskID) == "" ||
		strings.TrimSpace(tool) == "" || strings.TrimSpace(operationKey) == "" ||
		strings.TrimSpace(description) == "" {
		return domain.ErrBadRequest
	}

	s.externalMu.Lock()
	defer s.externalMu.Unlock()

	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.Source != externalLeadSource+client {
		return domain.ErrNotFound
	}

	// The correlation token is never returned to the connected app or thread.
	hash := sha256.Sum256([]byte(userID + "\x00" + client + "\x00" + taskID +
		"\x00" + tool + "\x00" + operationKey))
	eventKey := "external-mcp-" + hex.EncodeToString(hash[:16])

	_, exists, err := s.deps.Events.EventByToolUse(ctx, taskID, eventKey)
	if err != nil {
		return err
	}
	if exists {
		return nil // A client retry must not duplicate an observed action.
	}
	if task.Status == domain.StatusDone || task.Status == domain.StatusFailed {
		return domain.ErrWrongState
	}

	_, err = s.deps.Events.AppendEvent(ctx, taskID, domain.Event{
		Kind: domain.EventAction, Text: description, Agent: task.Source,
		ToolUseID: eventKey, Time: s.now(),
	})
	return err
}
