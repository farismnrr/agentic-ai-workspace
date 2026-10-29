package application

import (
	"context"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"strings"
	"testing"
	"time"
)

type scriptedCLI struct {
	seen      []CLIRequest
	responses []CLIResult
	events    [][]CLIEvent
}

func (*scriptedCLI) Ready(context.Context, string, string) (bool, error) { return true, nil }
func (f *scriptedCLI) Run(_ context.Context, req CLIRequest, emit func(CLIEvent)) (CLIResult, error) {
	i := len(f.seen)
	f.seen = append(f.seen, req)
	if i < len(f.events) {
		for _, e := range f.events[i] {
			emit(e)
		}
	}
	if i < len(f.responses) {
		return f.responses[i], nil
	}
	return CLIResult{Reason: "Unexpected retry"}, nil
}

type contextStore struct {
	TaskStore
	readings [][2]int64
	states   []string
}

func (s *contextStore) SetContextState(_ context.Context, _ string, state string) error {
	s.states = append(s.states, state)
	return nil
}
func (s *contextStore) SetContext(_ context.Context, _ string, used, window int64) error {
	s.readings = append(s.readings, [2]int64{used, window})
	return nil
}

type fakeEvents struct{ EventStore }

func (fakeEvents) EventsAfter(context.Context, string, int64) ([]domain.Event, error) {
	return []domain.Event{{Text: "Earlier decision: preserve changes"}}, nil
}
func (fakeEvents) AppendEvent(_ context.Context, _ string, e domain.Event) (domain.Event, error) {
	return e, nil
}
func simulateCLI(task domain.Task, f *scriptedCLI) (CLIResult, *planSink, bool, *contextStore) {
	store := &contextStore{}
	r := NewRunner(Deps{CLI: f, Tasks: store, Events: fakeEvents{}, Now: time.Now}, Limits{})
	result, sink, fresh, _ := r.runCLIWithRecovery(context.Background(), task, CLIRequest{Session: "old", TaskID: task.ID, Prompt: "Latest: do not delete"}, &planSink{runner: r, task: task})
	return result, sink, fresh, store
}
func TestCLIInitialFullRollover(t *testing.T) {
	fake := &scriptedCLI{responses: []CLIResult{{Session: "new"}}}
	_, _, fresh, store := simulateCLI(domain.Task{ID: "task", ContextUsed: 31000, ContextWindow: 32000}, fake)
	if !fresh || len(fake.seen) != 1 || fake.seen[0].Session != "" {
		t.Fatalf("preflight: %+v", fake.seen)
	}
	if !strings.Contains(fake.seen[0].Prompt, "Latest: do not delete") || !strings.Contains(fake.seen[0].Prompt, "Earlier decision") {
		t.Fatal("lost instructions")
	}
	if len(store.readings) != 1 || store.readings[0] != [2]int64{0, 0} {
		t.Fatalf("stale meter %+v", store.readings)
	}
	if len(store.states) < 2 || store.states[0] != "compacting" || store.states[len(store.states)-1] != "unknown" {
		t.Fatalf("fresh session state never settled: %+v", store.states)
	}
}
func TestCLIOverflowRetriesExactlyOnce(t *testing.T) {
	fake := &scriptedCLI{responses: []CLIResult{{Reason: "context window exceeded"}, {Session: "new"}}}
	result, _, fresh, _ := simulateCLI(domain.Task{ID: "task", ContextUsed: 1000, ContextWindow: 32000}, fake)
	if !fresh || result.Session != "new" || len(fake.seen) != 2 || fake.seen[0].Session != "old" || fake.seen[1].Session != "" {
		t.Fatalf("retry: %+v", fake.seen)
	}
}
func TestCLINeverReplaysCommandsAfterOverflow(t *testing.T) {
	fake := &scriptedCLI{responses: []CLIResult{{Reason: "context window exceeded"}}, events: [][]CLIEvent{{{Kind: CLICommand, Technical: "touch file"}}}}
	_, sink, fresh, _ := simulateCLI(domain.Task{ID: "task", ContextUsed: 1000, ContextWindow: 32000}, fake)
	if fresh || len(fake.seen) != 1 || !sink.didStep {
		t.Fatalf("unsafe replay %d", len(fake.seen))
	}
}
func TestCLIFreshOverflowStopsAfterOneRecovery(t *testing.T) {
	f := &scriptedCLI{responses: []CLIResult{{Reason: "context window exceeded"}, {Reason: "context window exceeded"}}}
	result, _, fresh, _ := simulateCLI(domain.Task{ID: "task", ContextUsed: 1000, ContextWindow: 32000}, f)
	if !fresh || len(f.seen) != 2 || result.Reason == "" {
		t.Fatalf("loop after overflow: calls=%d result=%+v", len(f.seen), result)
	}
}
func TestCLIUsageOnlyMeasuredWhenWindowKnown(t *testing.T) {
	f := &scriptedCLI{responses: []CLIResult{{}}, events: [][]CLIEvent{{{Kind: CLIUsage, Used: 24000, Window: 32000}}}}
	_, _, _, store := simulateCLI(domain.Task{ID: "task"}, f)
	if len(store.readings) != 2 || store.readings[1] != [2]int64{24000, 32000} {
		t.Fatalf("live meter %+v", store.readings)
	}
	f = &scriptedCLI{responses: []CLIResult{{}}, events: [][]CLIEvent{{{Kind: CLIUsage, Used: 24000, Window: 0}}}}
	_, _, _, store = simulateCLI(domain.Task{ID: "task"}, f)
	if len(store.readings) != 1 {
		t.Fatalf("unknown vendor window should not claim occupancy: %+v", store.readings)
	}
}
func TestCLINeverReplaysStartedButUnfinishedTool(t *testing.T) {
	f := &scriptedCLI{responses: []CLIResult{{Reason: "context window exceeded"}}, events: [][]CLIEvent{{{Kind: CLIActivity}}}}
	_, sink, fresh, _ := simulateCLI(domain.Task{ID: "task", ContextUsed: 1000, ContextWindow: 32000}, f)
	if fresh || len(f.seen) != 1 || !sink.didStep {
		t.Fatalf("replayed an unfinished tool: %d calls", len(f.seen))
	}
}

func TestCLIRecoveryAfterPreviousFailureUsesFreshSession(t *testing.T) {
	f := &scriptedCLI{responses: []CLIResult{{Session: "fresh"}}}
	_, _, fresh, _ := simulateCLI(domain.Task{ID: "task", ContextState: "failed"}, f)
	if !fresh || len(f.seen) != 1 || f.seen[0].Session != "" {
		t.Fatalf("did not recover a previously failed context: %+v", f.seen)
	}
}
