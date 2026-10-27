package sqlite

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestWorkCallRetryReturnsExistingChangeEvent(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-call-retry", "t-call-retry", "")
	call := domain.Event{
		Kind: domain.EventAction, Text: "edit page", Files: []string{"page.ts"}, Undo: []byte("undo-page"),
		ToolUseID: "call-unique", Work: &domain.WorkRef{ID: work.ID},
	}
	_, first, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, call)
	if err != nil {
		t.Fatal(err)
	}
	call.Text = "retry must not append a second change"
	_, replay, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, call)
	if err != nil || replay.ID != first.ID || replay.Text != first.Text {
		t.Fatalf("retry = %+v, want event %q (%v)", replay, first.ID, err)
	}
	events, err := store.EventsAfter(ctx, "t-call-retry", 0)
	if err != nil || len(events) != 2 || !events[1].CanUndo() {
		t.Fatalf("retry stored duplicate or lost undo: %d events, %v", len(events), err)
	}
}
