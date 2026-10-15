package application_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func read(id string, used int64) step {
	s := call(id, "read_file", map[string]string{"path": "a"})
	s.reply.Usage = domain.Usage{Input: used, Output: 100}

	return s
}

func text(words string) step {
	return step{reply: application.Reply{Blocks: []domain.Block{{Kind: domain.BlockText, Text: words}}}}
}

func TestALongTaskIsSummarisedAndKeepsGoing(t *testing.T) {
	t.Parallel()

	r := newRig(nil, read("c1", 30_000), text("SUMMARY of the work"), finish("Done"))
	r.windows.tokens = 50_000
	task := r.start(t, "Read a lot")
	r.run(task.ID)

	if len(r.model.requests) != 3 {
		t.Fatalf("want a turn, a summary and a turn, got %d calls", len(r.model.requests))
	}

	final := r.model.requests[2].Messages
	if len(final) != 1 || !final[0].Blocks[0].Summary || final[0].Blocks[0].Text == "" {
		t.Fatalf("the next turn must start from the summary alone: %+v", final)
	}

	if r.task(t, task.ID).Status == domain.StatusFailed {
		t.Fatal("compaction must not fail the task")
	}
}

func TestACumulativelyHugeTaskDoesNotStop(t *testing.T) {
	t.Parallel()

	steps := make([]step, 0, 6)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		steps = append(steps, read(id, 90_000))
	}

	r := newRig(nil, append(steps, finish("Done"))...)
	task := r.start(t, "Long but within the window")
	r.run(task.ID)

	if len(r.model.requests) != 6 || r.task(t, task.ID).Status == domain.StatusFailed {
		t.Fatalf("only the context size matters, not the total spent: %d calls", len(r.model.requests))
	}
}

func TestASummaryThatReachesForToolsIsSkipped(t *testing.T) {
	t.Parallel()

	r := newRig(nil, read("c1", 30_000), call("x", "read_file", map[string]string{"path": "b"}), finish("Done"))
	r.windows.tokens = 50_000
	task := r.start(t, "Read a lot")
	r.run(task.ID)

	for _, request := range r.model.requests[2:] {
		if request.Messages[0].Blocks[0].Summary {
			t.Fatal("a reply that is not a summary must not replace the conversation")
		}
	}
}
