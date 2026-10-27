package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func normalizePlanInput(in UpdatePlanInput) ([]domain.PlanStep, error) {
	if len(in.Steps) > domain.MaxWorkSteps {
		return nil, domain.ErrBadRequest
	}
	steps := make([]domain.PlanStep, 0, len(in.Steps))
	ids := map[string]bool{}
	for _, step := range in.Steps {
		clean, err := normalizePlanStep(step, ids)
		if err != nil {
			return nil, domain.ErrBadRequest
		}
		ids[clean.ID] = clean.ID != ""
		steps = append(steps, clean)
	}
	return steps, nil
}

func normalizePlanStep(step PlanStepInput, ids map[string]bool) (domain.PlanStep, error) {
	title := strings.TrimSpace(step.Title)
	id, _ := domain.Redact(step.ID)
	if title == "" || utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(step.Detail) > 1000 ||
		utf8.RuneCountInString(id) > 80 || !validPlanState(step.State) || (id != "" && ids[id]) {
		return domain.PlanStep{}, domain.ErrBadRequest
	}
	title, _ = domain.Redact(title)
	detail, _ := domain.Redact(step.Detail)
	if utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(detail) > 1000 {
		return domain.PlanStep{}, domain.ErrBadRequest
	}
	return domain.PlanStep{ID: id, Title: title, Detail: detail, State: step.State}, nil
}

func validPlanState(state domain.PlanState) bool {
	return state == "" || state == domain.PlanPending || state == domain.PlanActive ||
		state == domain.PlanDone || state == domain.PlanFailed
}

func requestHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func (w *Works) replay(
	ctx context.Context, work domain.ExternalWork, tool, key, hash string,
) (PlanResult, bool, error) {
	if key == "" {
		return PlanResult{}, false, nil
	}
	op, found, err := w.store.FindOp(ctx, work.ID, key)
	if err != nil {
		return PlanResult{}, false, fmt.Errorf("find work operation: %w", err)
	}
	if !found {
		return PlanResult{}, false, nil
	}
	if op.Tool != tool || op.ReqHash != hash {
		return PlanResult{}, false, domain.ErrKeyReused
	}
	var result PlanResult
	if err := json.Unmarshal([]byte(op.Response), &result); err != nil {
		return PlanResult{}, false, fmt.Errorf("decode saved work response: %w", err)
	}
	return result, true, nil
}

func (w *Works) commitPlan(ctx context.Context, work domain.ExternalWork, key, hash string,
	response PlanResult, patch *WorkPatch, task *TaskPatch, events []domain.Event,
) error {
	var op *WorkOp
	if key != "" {
		encoded, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("encode work plan response: %w", err)
		}
		op = &WorkOp{
			WorkID: work.ID, OpKey: key, Tool: "update_plan", ReqHash: hash,
			Response: string(encoded), At: w.now().Unix(),
		}
	}
	_, _, err := w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Op: op, Patch: patch, Task: task, Events: events,
	})
	if err != nil {
		return fmt.Errorf("update work plan: %w", err)
	}
	return nil
}
