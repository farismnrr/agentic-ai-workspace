package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestConcurrentApprovalCreatesOneDecisionAndHandoff(t *testing.T) {
	ctx, db, store := newConcurrentWorkStore(t)
	work := createTestWork(t, ctx, store, "w-race-approval", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for index := range 2 {
		wait.Add(1)
		go func(number int) {
			defer wait.Done()
			in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
			in.Decision.ID = fmt.Sprintf("decision-race-%d", number)
			in.Decision.IdemKey = fmt.Sprintf("race-key-%d", number)
			in.Decision.ReqHash = fmt.Sprintf("race-hash-%d", number)
			in.HandoffID = fmt.Sprintf("handoff-race-%d", number)
			_, err := store.DecidePlan(ctx, in)
			results <- err
		}(index)
	}
	wait.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, domain.ErrTryAgain) {
			t.Fatalf("concurrent approval error = %v", err)
		}
	}
	assertOneApprovalAndHandoff(t, ctx, db, work.ID)
}

func assertOneApprovalAndHandoff(t *testing.T, ctx context.Context, db *sql.DB, workID string) {
	t.Helper()
	var decisions, handoffs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_decisions WHERE work_id = ?`, workID).
		Scan(&decisions)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, workID).
		Scan(&handoffs)
	if decisions != 1 || handoffs != 1 {
		t.Fatalf("concurrent approvals persisted %d decisions and %d handoffs", decisions, handoffs)
	}
}
