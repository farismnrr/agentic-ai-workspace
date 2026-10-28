package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	workProgressLimit   = 10
	workProgressPreview = 320
	workDocumentPreview = 500
	workMinimalPreview  = 100
	workStateMaxBytes   = 8 * 1024
)

// State contains a bounded, read-only snapshot of an external work session.
type StateResult struct {
	Review       *domain.PlanReview `json:"review,omitempty"`
	NextAction   string             `json:"next_action"`
	Role         string             `json:"role,omitempty"`
	PRD          string             `json:"prdPreview,omitempty"`
	WorkID       string             `json:"workId"`
	Research     string             `json:"researchPreview,omitempty"`
	State        domain.WorkState   `json:"state"`
	PlanState    domain.PlanPhase   `json:"planState"`
	TaskID       string             `json:"-"`
	Questions    QuestionGroups     `json:"questions"`
	Plan         []domain.PlanStep  `json:"plan"`
	Progress     []string           `json:"progress"`
	Evidence     EvidenceCounts     `json:"evidence"`
	Revision     int64              `json:"revision"`
	PlanRevision int64              `json:"planRevision"`
	ResearchSize int                `json:"researchSize"`
	PRDSize      int                `json:"prdSize"`
	Stale        bool               `json:"stale"`
}

// QuestionGroups keeps both pending and resolved prompts available after reconnect.
type QuestionGroups struct {
	Open     []WorkQuestion `json:"open"`
	Answered []WorkQuestion `json:"answered"`
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
	work, task, events, err := w.readWorkSnapshot(ctx, userID, client, projectID, workID)
	if err != nil {
		return StateResult{}, err
	}
	state := stateSnapshot(work, task, events, w.now(), w.thresholds)
	state.Review, err = w.latestReview(ctx, work.ID)
	if err != nil {
		return StateResult{}, fmt.Errorf("read plan review state: %w", err)
	}
	state.Questions, err = w.questionGroups(ctx, work.ID)
	if err != nil {
		return StateResult{}, fmt.Errorf("read work questions: %w", err)
	}
	state.NextAction = stateNextAction(state, events, task.Plan)
	return boundedState(state), nil
}
