package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const externalLeadSource = "orchestrator:"

// CreateExternalLeadTask creates a visible external-led task WITHOUT enqueuing the internal lead.
// Idempotency is tied to the owner, connected client, project and caller's request key.
func (s *Service) CreateExternalLeadTask(ctx context.Context, userID, client, projectID, prompt, requestKey string) (domain.Task, error) {
	prompt, client, requestKey = strings.TrimSpace(prompt), strings.TrimSpace(client), strings.TrimSpace(requestKey)
	if prompt == "" || client == "" || len(requestKey) < 8 || len(requestKey) > 128 {
		return domain.Task{}, domain.ErrBadRequest
	}
	if _, err := s.deps.Projects.Name(ctx, userID, projectID); err != nil {
		return domain.Task{}, domain.ErrNoProject
	}
	sum := sha256.Sum256([]byte("external-lead\x00" + userID + "\x00" + client + "\x00" + projectID + "\x00" + requestKey))
	id := hex.EncodeToString(sum[:16])
	if existing, err := s.deps.Tasks.Find(ctx, userID, id); err == nil {
		if existing.ProjectID != projectID || existing.Source != externalLeadSource+client || existing.Summary != prompt {
			return domain.Task{}, domain.ErrBadRequest
		}
		return existing, nil
	}
	now := s.now()
	task := domain.Task{
		ID: id, ProjectID: projectID, UserID: userID, Title: titleOf(prompt), Summary: prompt,
		Source: externalLeadSource + client, Team: true, PlanFirst: true,
		ReviewMode: domain.ReviewModeSwift, Status: domain.StatusWorking,
		CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{},
	}
	event := domain.Event{Kind: domain.EventRequest, Text: prompt, Time: now}
	if err := s.deps.Tasks.CreateWithRequest(ctx, task, event, prompt); err != nil {
		// Concurrent idempotent creates can race on the same unique task ID.
		existing, getErr := s.deps.Tasks.Find(ctx, userID, id)
		if getErr == nil && existing.ProjectID == projectID && existing.Source == task.Source && existing.Summary == task.Summary {
			return existing, nil
		}
		return domain.Task{}, fmt.Errorf("create external-led task: %w", err)
	}
	return task, nil
}

// CompleteExternalLeadTask closes an external-led task, not any of its internal worker tasks.
func (s *Service) CompleteExternalLeadTask(ctx context.Context, userID, taskID, summary string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(task.Source, externalLeadSource) || task.Status == domain.StatusDone {
		return domain.ErrWrongState
	}
	if strings.TrimSpace(summary) == "" {
		return domain.ErrBadRequest
	}
	if s.externalActive(taskID) {
		return errors.New("worker jobs are still active")
	}
	// External completion is a report for human review, not verified proof.
	return s.deps.Tasks.Finish(ctx, taskID, summary, nil, s.now())
}
