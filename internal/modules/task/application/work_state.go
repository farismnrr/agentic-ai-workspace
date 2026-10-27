package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	workProgressLimit   = 10
	workProgressPreview = 320
	workDocumentPreview = 500
	workMinimalPreview  = 100
	workStateMaxBytes   = 8 * 1024
	workIdleAfter       = 30 * time.Minute
)

// State contains a bounded, read-only snapshot of an external work session.
//
//nolint:govet // fieldalignment: these fields preserve the stable state response schema.
type StateResult struct {
	TaskID       string            `json:"-"`
	Stale        bool              `json:"stale"`
	Plan         []domain.PlanStep `json:"plan"`
	Progress     []string          `json:"progress"`
	WorkID       string            `json:"workId"`
	Revision     int64             `json:"revision"`
	State        domain.WorkState  `json:"state"`
	Role         string            `json:"role,omitempty"`
	Research     string            `json:"researchPreview,omitempty"`
	PRD          string            `json:"prdPreview,omitempty"`
	Evidence     EvidenceCounts    `json:"evidence"`
	PlanRevision int64             `json:"planRevision"`
	ResearchSize int               `json:"researchSize"`
	PRDSize      int               `json:"prdSize"`
}

// EvidenceCounts separate reported progress from server observations and real runs.
type EvidenceCounts struct {
	Claimed  int `json:"claimed"`
	Observed int `json:"observed"`
	Ran      int `json:"ran"`
}

// State reads a work snapshot; silence is displayed as idle and never as completion.
func (w *Works) State(ctx context.Context, userID, client, projectID, workID string) (StateResult, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return StateResult{}, err
	}
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return StateResult{}, domain.ErrUnknownWork
	}
	task, err := w.deps.Tasks.Find(ctx, userID, work.TaskID)
	if err != nil || task.ProjectID != projectID || task.Source != client || task.WorkID != work.ID {
		return StateResult{}, domain.ErrUnknownWork
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return StateResult{}, fmt.Errorf("read work events: %w", err)
	}
	return stateSnapshot(work, task, events, w.now()), nil
}

func stateSnapshot(work domain.ExternalWork, task domain.Task, events []domain.Event, now time.Time) StateResult {
	state := StateResult{
		TaskID: work.TaskID, WorkID: work.ID, Revision: work.Rev, State: work.State, Role: work.RoleKey,
		Plan: task.Plan, PlanRevision: work.PlanRev, ResearchSize: len(task.Research), Research: preview(task.Research),
		PRDSize: len(task.PRD), PRD: preview(task.PRD), Progress: []string{},
		Stale: work.State == domain.WorkOpen && now.Sub(work.UpdatedAt) > workIdleAfter,
	}
	for _, event := range events {
		countEvidence(&state, event)
		if progress, ok := progressLine(work.ID, event); ok {
			state.Progress = append(state.Progress, progress)
		}
	}
	if len(state.Progress) > workProgressLimit {
		state.Progress = state.Progress[len(state.Progress)-workProgressLimit:]
	}
	return boundedState(state)
}

func countEvidence(state *StateResult, event domain.Event) {
	if event.Work == nil {
		return
	}
	switch event.Work.Evidence {
	case domain.EvidenceClaimed:
		state.Evidence.Claimed++
	case domain.EvidenceObserved:
		state.Evidence.Observed++
	case domain.EvidenceRan:
		state.Evidence.Ran++
	}
}

func progressLine(workID string, event domain.Event) (string, bool) {
	if event.Work == nil || event.Work.ID != workID || event.Kind == domain.EventRequest ||
		strings.TrimSpace(event.Text) == "" {
		return "", false
	}
	return truncateRunes(event.Text, workProgressPreview), true
}

func preview(text string) string { return truncateRunes(text, workDocumentPreview) }

func truncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

func boundedState(state StateResult) StateResult {
	for len(mustJSON(state)) > workStateMaxBytes {
		if len(state.Progress) > 0 {
			state.Progress = state.Progress[1:]
			continue
		}
		if len(state.Plan) > 0 {
			state.Plan = state.Plan[:len(state.Plan)-1]
			continue
		}
		state.Research = truncateRunes(state.Research, workMinimalPreview)
		state.PRD = truncateRunes(state.PRD, workMinimalPreview)
		break
	}
	return state
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
