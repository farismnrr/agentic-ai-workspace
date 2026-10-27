package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestSetRoleNormalizesAndRejectsOrchestrator(t *testing.T) {
	works, _, _ := newWorksTest(t)
	created, err := works.Begin(context.Background(), beginInput("request"))
	if err != nil {
		t.Fatal(err)
	}
	role, err := works.SetRole(context.Background(), "u1", "ChatGPT", "proj", created.WorkID, "dEvElOpEr")
	if err != nil || role.Key != "developer" {
		t.Fatalf("normalized role = %+v, %v", role, err)
	}
	_, err = works.SetRole(context.Background(), "u1", "ChatGPT", "proj", created.WorkID, "Orchestrator")
	if !errors.Is(err, domain.ErrBadRole) {
		t.Fatalf("orchestrator role error = %v", err)
	}
}

func TestWorkLookupIsScopedAndStateSurvivesServiceReopen(t *testing.T) {
	works, store, clock := newWorksTest(t)
	created, err := works.Begin(context.Background(), beginInput("request"))
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range []struct{ user, client, project string }{
		{"other", "ChatGPT", "proj"}, {"u1", "Gemini", "proj"}, {"u1", "ChatGPT", "other"},
	} {
		_, stateErr := works.State(context.Background(), identity.user, identity.client, identity.project, created.WorkID)
		if !errors.Is(stateErr, domain.ErrUnknownWork) {
			t.Fatalf("wrong identity %+v gave %v", identity, stateErr)
		}
	}
	prior := store.works[created.WorkID]
	prior.UpdatedAt = clock.Add(-31 * time.Minute)
	store.works[created.WorkID] = prior
	reopened := NewWorks(testDeps(store, clock), store)
	state, err := reopened.State(context.Background(), "u1", "ChatGPT", "proj", created.WorkID)
	if err != nil || !state.Stale || state.State != domain.WorkOpen {
		t.Fatalf("reopened state = %+v, %v", state, err)
	}
}

func TestWorksRateLimitsArePerUserAndPerOperationClass(t *testing.T) {
	works, _, _ := newWorksTest(t)
	result, err := works.Begin(context.Background(), beginInput("state work"))
	if err != nil {
		t.Fatal(err)
	}
	input := beginInput("same idempotent request")
	input.IdempotencyKey = "rate-limit-001"
	for i := 0; i < domain.BeginWorkPerMinute-1; i++ {
		if _, callErr := works.Begin(context.Background(), input); callErr != nil {
			t.Fatal(callErr)
		}
	}
	if _, callErr := works.Begin(context.Background(), input); !errors.Is(callErr, domain.ErrRateLimited) {
		t.Fatalf("begin rate limit error = %v", callErr)
	}
	for i := 0; i < domain.OtherWorkPerMinute; i++ {
		if _, callErr := works.State(context.Background(), "u1", "ChatGPT", "proj", result.WorkID); callErr != nil {
			t.Fatal(callErr)
		}
	}
	_, err = works.State(context.Background(), "u1", "ChatGPT", "proj", result.WorkID)
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("state rate limit error = %v", err)
	}
}

func TestBeginAbandonsOpenWorkAfter24Hours(t *testing.T) {
	works, store, clock := newWorksTest(t)
	first, err := works.Begin(context.Background(), beginInput("first"))
	if err != nil {
		t.Fatal(err)
	}
	prior := store.works[first.WorkID]
	prior.UpdatedAt = clock.Add(-25 * time.Hour)
	store.works[first.WorkID] = prior
	if _, err := works.Begin(context.Background(), beginInput("second")); err != nil {
		t.Fatal(err)
	}
	taskID := store.works[first.WorkID].TaskID
	if store.works[first.WorkID].State != domain.WorkAbandoned || store.tasks[taskID].Status != domain.StatusFailed {
		t.Fatalf("stale work not abandoned: work=%+v task=%+v", store.works[first.WorkID], store.tasks[taskID])
	}
}
