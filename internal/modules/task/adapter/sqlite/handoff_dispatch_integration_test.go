package sqlite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestApprovedSnapshotDispatchesThroughOrchestratorWithoutExternalCompletionPower(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	at := time.Unix(40, 0)
	work := createTestWork(t, ctx, store, "w-dispatch", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	write := approvalWrite(work, revision.RevNo, at)
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET plan='[{"title":"mutated"}]',
		prd='mutated requirements', research='mutated research' WHERE id = ?`, work.TaskID); err != nil {
		t.Fatal(err)
	}
	queue := &handoffQueueFake{}
	ids := &workTestIDs{}
	deps := taskapp.Deps{
		Tasks: store, LinkedTasks: store, Events: store, Transcript: store, Models: handoffModelSource{},
		IDs: ids, Now: func() time.Time { return at },
	}
	service := taskapp.NewService(deps, queue)
	handoffs := taskapp.NewHandoffs(service, store)
	internalID, err := handoffs.Kick(ctx, write.HandoffID)
	queued := queue.snapshot()
	if err != nil || internalID == "" || len(queued) != 1 || queued[0] != internalID {
		t.Fatalf("handoff kick = %q queue=%v err=%v", internalID, queued, err)
	}
	internal, err := store.Find(ctx, work.UserID, internalID)
	if err != nil || internal.OriginTaskID != work.TaskID || internal.WorkID != "" || internal.Source != "" ||
		internal.PlanFirst || internal.Status != domain.StatusQueued {
		t.Fatalf("Orchestrator task = %+v, %v", internal, err)
	}
	transcript, err := store.LoadTranscript(ctx, internalID)
	if err != nil || len(transcript) != 1 || transcript[0].Blocks[0].Text == "" {
		t.Fatalf("handoff prompt not persisted: %+v, %v", transcript, err)
	}
	text := transcript[0].Blocks[0].Text
	if !strings.Contains(text, "requirements snapshot") || !strings.Contains(text, "research snapshot") ||
		!strings.Contains(text, "answers") || strings.Contains(text, "mutated") {
		t.Fatalf("handoff prompt escaped the approved snapshot: %s", text)
	}
	works := taskapp.NewWorks(deps, store, store, store)
	if _, completionErr := works.Complete(ctx, work.UserID, work.Client, work.ProjectID,
		work.ID, "completed", "External done", nil); !errors.Is(completionErr, domain.ErrWorkClosed) {
		t.Fatalf("external completion changed approved work: %v", completionErr)
	}
	internal, err = store.Find(ctx, work.UserID, internalID)
	if err != nil || internal.Status != domain.StatusQueued {
		t.Fatalf("external completion changed internal task: %+v, %v", internal, err)
	}
}

func TestConcurrentAppAndMCPApprovalDispatchOneInternalTask(t *testing.T) {
	ctx, db, store := newConcurrentWorkStore(t)
	at := time.Unix(50, 0)
	work := createTestWork(t, ctx, store, "w-concurrent-dispatch", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	queue := &handoffQueueFake{}
	deps := taskapp.Deps{
		Tasks: store, LinkedTasks: store, Events: store, Transcript: store, Models: handoffModelSource{},
		IDs: &handoffIDs{}, Now: func() time.Time { return at },
	}
	service := taskapp.NewService(deps, queue)
	handoffs := taskapp.NewHandoffs(service, store)
	reviews := taskapp.NewPlanReviews(store, &handoffIDs{}, func() time.Time { return at }, handoffs, nil)
	in := taskapp.PlanReviewInput{
		ProjectID: work.ProjectID, TaskID: work.TaskID, WorkID: work.ID,
		Revision: revision.RevNo, IdempotencyKey: "same-approval-key",
	}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	wait.Add(2)
	go func() {
		defer wait.Done()
		_, err := reviews.ApproveFromApp(ctx, work.UserID, in)
		results <- err
	}()
	go func() {
		defer wait.Done()
		_, err := reviews.ApproveFromMCP(ctx, work.UserID, work.Client, in)
		results <- err
	}()
	wait.Wait()
	close(results)
	approved := 0
	for err := range results {
		if err == nil {
			approved++
		} else if !errors.Is(err, domain.ErrTryAgain) {
			t.Fatalf("concurrent approval error = %v", err)
		}
	}
	if approved == 0 {
		t.Fatal("both app and MCP approvals failed")
	}
	var internalTasks, handoffRows int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE origin_task_id = ?`, work.TaskID).
		Scan(&internalTasks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, work.ID).
		Scan(&handoffRows)
	if internalTasks != 1 || handoffRows != 1 {
		t.Fatalf("concurrent approvals created %d internal tasks and %d handoffs", internalTasks, handoffRows)
	}
}

type handoffModelSource struct{}

func (handoffModelSource) Client(context.Context, string, string) (taskapp.Model, error) {
	return handoffModel{}, nil
}

func (handoffModelSource) Default(context.Context, string) (string, string, error) {
	return "provider", "model", nil
}

func (handoffModelSource) DefaultEffort(context.Context, string) domain.Effort {
	return domain.EffortDefault
}

func (handoffModelSource) Tier(context.Context, string, string) (string, string, domain.Effort, bool) {
	return "", "", "", false
}

type handoffModel struct{}

func (handoffModel) Complete(context.Context, taskapp.Request) (taskapp.Reply, error) {
	return taskapp.Reply{}, nil
}

type handoffQueueFake struct {
	enqueued []string
	mu       sync.Mutex
}

func (f *handoffQueueFake) Enqueue(taskID, _ string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enqueued = append(f.enqueued, taskID)
}
func (*handoffQueueFake) Cancel(string) taskapp.CancelState { return taskapp.CancelNone }

func (f *handoffQueueFake) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.enqueued...)
}

type handoffIDs struct {
	mu sync.Mutex
	n  int
}

func (i *handoffIDs) ID() (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.n++
	return fmt.Sprintf("approval-id-%d", i.n), nil
}
