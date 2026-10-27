package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestCompleteIsIdempotentAndKeepsClaimsUnverified(t *testing.T) {
	works, store, begun := begunWork(t)
	ctx := context.Background()
	result, err := works.Complete(ctx, "u1", "ChatGPT", "proj", begun.WorkID,
		"completed", "Ready", []string{"token=secret-value"})
	if err != nil || result.Verified.Proven || result.Verified.Runs != 0 {
		t.Fatalf("claims were treated as verified: %+v, %v", result, err)
	}
	assertUnverifiedClaim(t, store, begun.WorkID)
	eventCount := len(store.events[store.works[begun.WorkID].TaskID])
	assertCompletionRetry(t, works, begun.WorkID, eventCount, store)
	assertClosedOutcome(t, works, begun.WorkID)
}

func assertUnverifiedClaim(t *testing.T, store *workFakeStore, workID string) {
	t.Helper()
	proof := store.tasks[store.works[workID].TaskID].Proof
	if proof == nil || len(proof.Checks) != 1 || proof.Checks[0].Verified ||
		!strings.Contains(proof.Checks[0].Text, "[redacted]") {
		t.Fatalf("unverified claims were promoted or not redacted: %+v", proof)
	}
}

func assertCompletionRetry(t *testing.T, works *Works, workID string, count int, store *workFakeStore) {
	t.Helper()
	_, err := works.Complete(context.Background(), "u1", "ChatGPT", "proj", workID,
		"completed", "Ready", nil)
	if err != nil || len(store.events[store.works[workID].TaskID]) != count {
		t.Fatalf("same completion was not idempotent: %v", err)
	}
}

func assertClosedOutcome(t *testing.T, works *Works, workID string) {
	t.Helper()
	_, err := works.Complete(context.Background(), "u1", "ChatGPT", "proj", workID,
		"failed", "Different", nil)
	if !errors.Is(err, domain.ErrWorkClosed) {
		t.Fatalf("terminal work accepted a different outcome: %v", err)
	}
}

func TestCompleteReportsServerRunsAndFileChanges(t *testing.T) {
	works, store, begun := begunWork(t)
	taskID := store.works[begun.WorkID].TaskID
	store.events[taskID] = append(store.events[taskID], domain.Event{
		Kind: domain.EventAction, Files: []string{"page.svelte"},
	}, domain.Event{
		Kind: domain.EventAction, Run: &domain.RunInfo{Name: "go test", ExitCode: 0},
	})
	result, err := works.Complete(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		"completed", "Ready", nil)
	if err != nil || !result.Verified.Proven || result.Verified.FilesChanged != 1 ||
		result.Verified.Runs != 1 || !result.Verified.LastRunPassed {
		t.Fatalf("server evidence was not returned accurately: %+v, %v", result, err)
	}
}

func TestBlockedWorkResumesWithProgress(t *testing.T) {
	works, store, begun := begunWork(t)
	ctx := context.Background()
	blockWork(t, works, begun.WorkID)
	if err := works.Progress(ctx, "u1", "ChatGPT", "proj", begun.WorkID, "Continuing", nil); err != nil {
		t.Fatal(err)
	}
	assertWorkResumed(t, store, begun.WorkID)
}

func TestRoleChangeResumesAndStopClosesWork(t *testing.T) {
	works, store, begun := begunWork(t)
	ctx := context.Background()
	blockWork(t, works, begun.WorkID)
	if _, err := works.SetRole(ctx, "u1", "ChatGPT", "proj", begun.WorkID, "Developer"); err != nil {
		t.Fatal(err)
	}
	assertWorkResumed(t, store, begun.WorkID)
	if err := works.Stop(ctx, "u1", "ChatGPT", "proj", begun.WorkID); err != nil {
		t.Fatal(err)
	}
	assertWorkStopped(t, store, begun.WorkID)
}

func blockWork(t *testing.T, works *Works, workID string) {
	t.Helper()
	if _, err := works.Complete(context.Background(), "u1", "ChatGPT", "proj", workID,
		"blocked", "Need an answer", nil); err != nil {
		t.Fatal(err)
	}
}

func assertWorkResumed(t *testing.T, store *workFakeStore, workID string) {
	t.Helper()
	work := store.works[workID]
	task := store.tasks[work.TaskID]
	if work.State != domain.WorkOpen || task.Status != domain.StatusWorking {
		t.Fatalf("blocked work did not reopen: work=%+v task=%+v", work, task)
	}
	last := store.events[task.ID][len(store.events[task.ID])-2]
	if last.Work == nil || last.Work.Phase != "resumed" {
		t.Fatalf("resume message missing: %+v", store.events[task.ID])
	}
}

func assertWorkStopped(t *testing.T, store *workFakeStore, workID string) {
	t.Helper()
	work := store.works[workID]
	task := store.tasks[work.TaskID]
	events := store.events[work.TaskID]
	last := events[len(events)-1]
	if work.State != domain.WorkStopped || task.Status != domain.StatusFailed ||
		last.Text != "Stopped by you" || last.Kind != domain.EventError {
		t.Fatalf("stop did not close work truthfully: work=%+v task=%+v event=%+v", work, task, last)
	}
}
