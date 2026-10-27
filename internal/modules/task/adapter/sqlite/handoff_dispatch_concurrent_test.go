package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
	assertConcurrentApprovalSucceeds(t, ctx, reviews, work, in)
	assertSingleDispatchedHandoff(t, ctx, db, work)
}

func assertConcurrentApprovalSucceeds(t *testing.T, ctx context.Context, reviews *taskapp.PlanReviews,
	work domain.ExternalWork, in taskapp.PlanReviewInput,
) {
	t.Helper()
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
	assertAtLeastOneApproval(t, results)
}

func assertAtLeastOneApproval(t *testing.T, results <-chan error) {
	t.Helper()
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
}

func assertSingleDispatchedHandoff(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork) {
	t.Helper()
	var internalTasks, handoffRows int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE origin_task_id = ?`, work.TaskID).
		Scan(&internalTasks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, work.ID).
		Scan(&handoffRows)
	if internalTasks != 1 || handoffRows != 1 {
		t.Fatalf("concurrent approvals created %d internal tasks and %d handoffs", internalTasks, handoffRows)
	}
}
