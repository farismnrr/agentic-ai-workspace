package application_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestAModelThatSaysTheRequestIsTooLongGetsASummaryAndAnotherTry(t *testing.T) {
	t.Parallel()

	tooLong := failing("This model's maximum context length is 128000 tokens", false)
	r := newRig(nil, read("c1", 1_000), tooLong, text("SUMMARY of the work"), finish("Done"))
	task := r.start(t, "Read a lot")
	r.run(task.ID)

	if len(r.model.requests) != 4 {
		t.Fatalf("want a turn, the refused turn, a summary and the turn again, got %d", len(r.model.requests))
	}

	again := r.model.requests[3].Messages
	if len(again) != 1 || !again[0].Blocks[0].Summary {
		t.Fatalf("the second try must start from the summary alone: %+v", again)
	}

	if r.task(t, task.ID).Status == domain.StatusFailed {
		t.Fatal("a window that was guessed too big must not fail the task")
	}
}

func TestAnErrorThatIsNotAboutLengthStillFailsTheTask(t *testing.T) {
	t.Parallel()

	r := newRig(nil, failing("invalid api key", false))
	task := r.start(t, "Say hi")
	r.run(task.ID)

	if len(r.model.requests) != 1 || r.task(t, task.ID).Status != domain.StatusFailed {
		t.Fatalf("want one call and a failed task, got %d calls", len(r.model.requests))
	}
}
