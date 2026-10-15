package application_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// create is what the composer sends for a task in the test project.
func create(prompt string, effort domain.Effort) application.CreateInput {
	return application.CreateInput{ProjectID: "proj", Prompt: prompt, Effort: effort}
}

func TestATaskKeepsTheChosenEffortOrThePersonsDefault(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	r.models.effort = domain.EffortMedium

	cases := []struct{ asked, want domain.Effort }{
		{domain.EffortHigh, domain.EffortHigh},
		{domain.EffortDefault, domain.EffortMedium},
	}

	for _, c := range cases {
		task, err := r.service.Create(t.Context(), "u1", create("Fix it", c.asked))
		if err != nil || task.Effort != c.want {
			t.Fatalf("asked %q: got %q, %v; want %q", c.asked, task.Effort, err, c.want)
		}
	}
}

func TestAnUnknownEffortIsRefusedBeforeAnythingIsSaved(t *testing.T) {
	t.Parallel()

	r := newRig(nil)

	_, err := r.service.Create(t.Context(), "u1", create("Fix it", "turbo"))
	if !errors.Is(err, domain.ErrBadEffort) {
		t.Fatalf("want ErrBadEffort, got %v", err)
	}
}

func TestTheRunnerAsksTheModelAtTheTasksEffort(t *testing.T) {
	t.Parallel()

	r := newRig(nil, finish("Done"))
	task, err := r.service.Create(t.Context(), "u1", create("Fix it", domain.EffortLow))
	if err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	if len(r.model.requests) == 0 || r.model.requests[0].Effort != domain.EffortLow {
		t.Fatalf("the request must carry the task's effort: %+v", r.model.requests)
	}
}
