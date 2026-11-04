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

// ContinueExternalLeadTask resumes a stopped or finished external-led task.
func (s *Service) ContinueExternalLeadTask(ctx context.Context, userID, taskID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(task.Source, externalLeadSource) {
		return domain.ErrWrongState
	}
	if task.Status != domain.StatusDone && task.Status != domain.StatusFailed {
		return domain.ErrWrongState
	}
	if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, s.now()); err != nil {
		return err
	}
	_, err = s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventMessage, Text: "Task continued by the external lead.", Time: s.now(),
	})
	return err
}

// CompleteExternalLeadTask closes an external-led task, not any of its internal worker tasks.
func (s *Service) CompleteExternalLeadTask(ctx context.Context, userID, taskID, summary string, outcome ...string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(task.Source, externalLeadSource) {
		return domain.ErrWrongState
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return domain.ErrBadRequest
	}
	out := ""
	if len(outcome) > 0 {
		out = strings.ToLower(strings.TrimSpace(outcome[0]))
	}
	if out == "pass" {
		verified := false
		if reader, ok := s.deps.Delegations.(interface {
			ListDelegations(context.Context, string) ([]Delegation, error)
		}); ok {
			if delegations, err := reader.ListDelegations(ctx, taskID); err == nil {
				for _, d := range delegations {
					if (d.RoleKey == "qa" || d.RoleKey == "reviewer" || d.RoleKey == "devops") && d.Status == DelegationDone {
						verified = true
						break
					}
				}
			}
		}
		if !verified {
			return errors.New("cannot report PASS without completed build or QA verification; use 'partial' or 'blocked'")
		}
	}
	if out != "" && !strings.HasPrefix(strings.ToLower(summary), out) && !strings.HasPrefix(strings.ToLower(summary), "["+out+"]") {
		summary = fmt.Sprintf("[%s] %s", strings.ToUpper(out), summary)
	}
	s.externalMu.Lock()
	defer s.externalMu.Unlock()
	if s.externalActiveLocked(taskID) {
		return errors.New("worker jobs are still active")
	}
	return s.deps.Tasks.CompleteExternalLead(ctx, taskID, task.Source, summary, s.now())
}
