package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestPlanReviewPathsShareDecisionServiceAndAuthenticatedIdentity(t *testing.T) {
	store := &planDecisionFake{}
	kicker := &handoffKickFake{}
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	reviews := NewPlanReviews(store, &workIDs{}, func() time.Time { return at }, kicker, nil)
	in := PlanReviewInput{
		ProjectID: "project-1", TaskID: "external-1", WorkID: "work-1", Revision: 3,
		IdempotencyKey: "approval-key-123",
	}
	appResult, err := reviews.ApproveFromApp(context.Background(), "user-1", in)
	if err != nil || store.write.Decision.Source != "app" || store.write.Decision.Client != "" ||
		appResult.HandoffID == "" || appResult.InternalTaskID != "internal-1" {
		t.Fatalf("app approval = %+v write=%+v err=%v", appResult, store.write, err)
	}
	mcpResult, err := reviews.ApproveFromMCP(context.Background(), "user-1", "ChatGPT", in)
	if err != nil || store.write.Decision.Source != "mcp" || store.write.Decision.Client != "ChatGPT" ||
		mcpResult.HandoffID != appResult.HandoffID {
		t.Fatalf("MCP approval = %+v write=%+v err=%v", mcpResult, store.write, err)
	}
	if kicker.calls != 2 {
		t.Fatalf("handoff kicks = %d, want one per committed approval/replay", kicker.calls)
	}
	if _, err := reviews.ApproveFromApp(context.Background(), "", in); !errors.Is(err, domain.ErrUnverifiedUser) {
		t.Fatalf("missing identity error = %v", err)
	}
}

func TestRequestChangesUsesTheDecisionStoreWithoutCreatingHandoff(t *testing.T) {
	store := &planDecisionFake{}
	kicker := &handoffKickFake{}
	reviews := NewPlanReviews(store, &workIDs{}, time.Now, kicker, nil)
	result, err := reviews.RequestChanges(context.Background(), "user-1", PlanReviewInput{
		ProjectID: "project-1", TaskID: "external-1", WorkID: "work-1", Revision: 2,
		IdempotencyKey: "changes-key-123", Note: "Clarify the first step.",
	})
	if err != nil || result.Decision.Kind != domain.PlanDecisionChangesRequested ||
		store.write.Decision.Source != "app" || store.write.HandoffID != "" || kicker.calls != 0 {
		t.Fatalf("request changes = %+v write=%+v kicks=%d err=%v", result, store.write, kicker.calls, err)
	}
}

type planDecisionFake struct {
	result *PlanDecisionResult
	write  PlanDecisionWrite
}

func (f *planDecisionFake) DecidePlan(_ context.Context, in PlanDecisionWrite) (PlanDecisionResult, error) {
	f.write = in
	if f.result != nil {
		return *f.result, nil
	}
	result := PlanDecisionResult{
		Decision: in.Decision, HandoffID: in.HandoffID, HandoffState: "pending",
	}
	f.result = &result
	return result, nil
}

type handoffKickFake struct{ calls int }

func (f *handoffKickFake) Kick(_ context.Context, _ string) (string, error) {
	f.calls++
	return "internal-1", nil
}
