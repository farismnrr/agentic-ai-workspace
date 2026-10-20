package application_test

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestAConnectedAppsThreadWorksWhileACallIsUnderWayAndIsDoneAfter(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	ctx := context.Background()
	in := application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT",
		Event: domain.Event{Kind: domain.EventAction, Text: "Running: npm install"},
	}

	id, err := r.service.Begin(ctx, "u1", in)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = r.service.Begin(ctx, "u1", in); err != nil { // a second call at the same time
		t.Fatal(err)
	}

	if got := r.task(t, id).Status; got != domain.StatusWorking {
		t.Fatalf("a call under way shows the thread as working, got %s", got)
	}

	r.service.End(ctx, id)

	if got := r.task(t, id).Status; got != domain.StatusWorking {
		t.Fatalf("one call is still under way, got %s", got)
	}

	r.service.End(ctx, id)

	if got := r.task(t, id).Status; got != domain.StatusDone {
		t.Fatalf("after the last call the thread is done again, got %s", got)
	}

	if kinds := r.kinds(id); len(kinds) != 2 {
		t.Fatalf("each call wrote its step: %v", kinds)
	}
}

func TestARestartMarksAConnectedAppsThreadDoneNotFailed(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	ctx := context.Background()

	id, err := r.service.Begin(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "Gemini", Event: domain.Event{Kind: domain.EventAction, Text: "Running: a"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = r.service.Recover(ctx); err != nil {
		t.Fatal(err)
	}

	if got := r.task(t, id).Status; got != domain.StatusDone {
		t.Fatalf("nothing was running for a connected app's thread, got %s", got)
	}
}
