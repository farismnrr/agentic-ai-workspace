package application

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// VerifiedWork summarizes evidence observed by the server, separate from assistant claims.
type VerifiedWork struct {
	FilesChanged  int  `json:"files_changed"`
	Runs          int  `json:"runs"`
	LastRunPassed bool `json:"last_run_passed"`
	Proven        bool `json:"proven"`
}

// Completion is the durable result and server-verified evidence returned to the caller.
type Completion struct {
	Outcome  string       `json:"outcome"`
	Verified VerifiedWork `json:"verified"`
	Already  bool         `json:"already,omitempty"`
}

// Complete closes or blocks one work session and derives proof only from server events.
func (w *Works) Complete(ctx context.Context, userID, client, projectID, workID, outcome, summary string,
	claims []string,
) (Completion, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return Completion{}, err
	}
	cleanSummary, cleanClaims, err := completionInput(outcome, summary, claims)
	if err != nil {
		return Completion{}, err
	}
	work, task, err := w.loadWorkTask(ctx, userID, client, projectID, workID)
	if err != nil {
		return Completion{}, err
	}
	if outcome == "completed" && (work.PlanState == domain.PlanReadyForApproval ||
		work.PlanState == domain.PlanAwaitingClarification) {
		return Completion{}, domain.ErrAwaitingPerson
	}
	if completionMatches(work, outcome) {
		result := completionResult(outcome, task.Proof)
		result.Already = true
		return result, nil
	}
	return w.finishWork(ctx, work, task, outcome, cleanSummary, cleanClaims)
}

func (w *Works) finishWork(ctx context.Context, work domain.ExternalWork, task domain.Task,
	outcome, summary string, claims []string,
) (Completion, error) {
	if work.State != domain.WorkOpen && work.State != domain.WorkBlocked {
		return Completion{}, domain.ErrWorkClosed
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return Completion{}, fmt.Errorf("read work evidence: %w", err)
	}
	proof := buildProof(events, claims)
	state, status, kind, text := completionState(outcome, summary)
	ended := w.now()
	event := workEvent(work, kind, text, "", ended)
	event.Work.Outcome = outcome
	taskPatch := &TaskPatch{Status: &status, Summary: &summary, Proof: proof}
	patch := &WorkPatch{State: &state}
	if state != domain.WorkBlocked {
		patch.EndedAt = &ended
	}
	if err := w.appendMutation(ctx, work, taskPatch, patch, []domain.Event{event}); err != nil {
		return Completion{}, err
	}
	return completionResult(outcome, proof), nil
}

func (w *Works) Stop(ctx context.Context, userID, client, projectID, workID string) error {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return err
	}
	work, _, err := w.loadWorkTask(ctx, userID, client, projectID, workID)
	if err != nil {
		return err
	}
	if work.State != domain.WorkOpen && work.State != domain.WorkBlocked {
		return domain.ErrWorkClosed
	}
	state, status, ended := domain.WorkStopped, domain.StatusFailed, w.now()
	event := workEvent(work, domain.EventError, "Stopped by you", "", ended)
	event.Work.Outcome = string(state)
	return w.appendMutation(ctx, work, &TaskPatch{Status: &status},
		&WorkPatch{State: &state, EndedAt: &ended}, []domain.Event{event})
}

func (w *Works) loadWorkTask(ctx context.Context, userID, client, projectID, workID string) (
	domain.ExternalWork, domain.Task, error,
) {
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return domain.ExternalWork{}, domain.Task{}, domain.ErrUnknownWork
	}
	task, err := w.deps.Tasks.Find(ctx, userID, work.TaskID)
	if err != nil || task.ProjectID != projectID || task.Source != client || task.WorkID != work.ID {
		return domain.ExternalWork{}, domain.Task{}, domain.ErrUnknownWork
	}
	return work, task, nil
}

func completionInput(outcome, summary string, claims []string) (string, []string, error) {
	if !validOutcome(outcome) {
		return "", nil, domain.ErrBadRequest
	}
	clean, err := sanitizeText(summary, domain.MaxWorkSummary)
	if err != nil {
		return "", nil, err
	}
	if clean == "" || utf8.RuneCountInString(summary) > domain.MaxWorkSummary || len(claims) > domain.MaxWorkClaims {
		return "", nil, domain.ErrBadRequest
	}
	cleanClaims, err := sanitizeClaims(claims)
	if err != nil {
		return "", nil, err
	}
	return clean, cleanClaims, nil
}

func validOutcome(outcome string) bool {
	return outcome == "completed" || outcome == "failed" || outcome == "blocked"
}

func sanitizeClaims(claims []string) ([]string, error) {
	cleanClaims := make([]string, 0, len(claims))
	for _, claim := range claims {
		value, claimErr := sanitizeText(claim, domain.MaxWorkClaim)
		if claimErr != nil || value == "" || utf8.RuneCountInString(claim) > domain.MaxWorkClaim {
			return nil, domain.ErrBadRequest
		}
		cleanClaims = append(cleanClaims, value)
	}
	return cleanClaims, nil
}

func completionMatches(work domain.ExternalWork, outcome string) bool {
	return (outcome == "completed" && work.State == domain.WorkCompleted) ||
		(outcome == "failed" && work.State == domain.WorkFailed) ||
		(outcome == "blocked" && work.State == domain.WorkBlocked)
}

func completionState(outcome, summary string) (domain.WorkState, domain.Status, domain.EventKind, string) {
	switch outcome {
	case "completed":
		return domain.WorkCompleted, domain.StatusDone, domain.EventReceipt, summary
	case "failed":
		return domain.WorkFailed, domain.StatusFailed, domain.EventError, summary
	default:
		return domain.WorkBlocked, domain.StatusNeedsInput, domain.EventMessage, summary
	}
}

func completionResult(outcome string, proof *domain.Proof) Completion {
	result := Completion{Outcome: outcome}
	if proof == nil {
		return result
	}
	result.Verified.Proven = proof.Proven
	for _, check := range proof.Checks {
		if check.Verified {
			result.Verified.Runs++
			result.Verified.LastRunPassed = check.Passed
		}
	}
	for _, row := range proof.Rows {
		if row.Label == "What changed" {
			_, _ = fmt.Sscanf(row.Value, "%d files changed", &result.Verified.FilesChanged)
			if row.Value == "1 file changed" {
				result.Verified.FilesChanged = 1
			}
		}
	}
	return result
}
