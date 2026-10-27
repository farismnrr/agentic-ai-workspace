package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTerminalWorkChangesBypassEventLimit(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	cases := []struct {
		name  string
		state domain.WorkState
	}{
		{name: "completed", state: domain.WorkCompleted},
		{name: "stopped", state: domain.WorkStopped},
		{name: "swept", state: domain.WorkAbandoned},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskID := "t-cap-" + tc.name
			work := createTestWork(t, ctx, store, "w-cap-"+tc.name, taskID, "")
			fillWorkEventLimit(t, ctx, store, taskID)
			ended := time.Now()
			closedEvent := domain.Event{
				Kind: domain.EventReceipt, Text: "closed",
				Work: &domain.WorkRef{ID: work.ID, Outcome: string(tc.state)},
			}
			_, _, err := store.Commit(ctx, application.WorkChange{
				WorkID: work.ID, UserID: "u", ExpectRev: work.Rev,
				Patch:  &application.WorkPatch{State: &tc.state, EndedAt: &ended},
				Events: []domain.Event{closedEvent},
			})
			if err != nil {
				t.Fatalf("terminal commit at event cap failed: %v", err)
			}
		})
	}
}

func fillWorkEventLimit(t *testing.T, ctx context.Context, store *Store, taskID string) {
	t.Helper()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for range domain.MaxWorkEvents - 1 {
		if _, err := insertEvent(ctx, tx, store.ids, taskID, domain.Event{Kind: domain.EventAction}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
