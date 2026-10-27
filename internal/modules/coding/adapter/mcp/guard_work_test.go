package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWorkFileEventOwnsUndoAndContainsChangedPaths(t *testing.T) {
	calls := &fakeWorkCalls{}
	activity := &captureRecorder{}
	g := &guard{workCalls: calls, recorder: activity}
	entry := Entry{UserID: "u", Client: "ChatGPT", Project: "p", Summary: "move old to new", Undo: []byte("blob")}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{WorkID: "w"})
	if err := g.succeeded(ctx, domain.ToolMoveFile, &entry, report{
		detail: "moved", undo: []byte("blob"), files: []string{"old", "new"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(calls.recorded) != 1 || len(calls.recorded[0].Files) != 2 ||
		calls.recorded[0].Evidence != "observed" || string(calls.recorded[0].Undo) != "blob" {
		t.Fatalf("work call = %+v", calls.recorded)
	}
	if len(activity.entries) != 1 || activity.entries[0].WorkID != "w" || activity.entries[0].Undo != nil ||
		!strings.Contains(activity.entries[0].Detail, "undo from the task's Changes tab") {
		t.Fatalf("activity row = %+v", activity.entries)
	}
}

var errWorkCallTestFailure = errors.New("storage unavailable")

func TestWorkFileEventFailureRetainsActivityUndoAndWarns(t *testing.T) {
	calls := &fakeWorkCalls{recordErr: errWorkCallTestFailure}
	activity := &captureRecorder{}
	g := &guard{workCalls: calls, recorder: activity}
	entry := Entry{UserID: "u", Client: "ChatGPT", Project: "p", Summary: "write page", Undo: []byte("blob")}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{WorkID: "w"})
	if err := g.succeeded(ctx, domain.ToolWriteFile, &entry, report{
		detail: "written", undo: []byte("blob"), files: []string{"page"},
	}); err != nil {
		t.Fatal(err)
	}
	if calls.attempts != 2 || len(activity.entries) != 1 ||
		string(activity.entries[0].Undo) != "blob" ||
		!strings.Contains(activity.entries[0].Detail, "recorded:false") ||
		len(calls.recorded) != 2 || calls.recorded[0].CallID == "" ||
		calls.recorded[0].CallID != calls.recorded[1].CallID {
		t.Fatalf("attempts=%d activity=%+v", calls.attempts, activity.entries)
	}
}

func TestWorkCommandUsesOneRunningEventAndRealExitCode(t *testing.T) {
	calls := &fakeWorkCalls{}
	g := &guard{workCalls: calls, recorder: &captureRecorder{}}
	entry := Entry{UserID: "u", Client: "Gemini", Project: "p", Summary: "command: go test"}
	state := g.newState(t.Context(), domain.ToolRunCommand, &entry, &mcp.CallToolRequest{}, "work-1")
	state.begin("p")
	ctx := context.WithValue(t.Context(), identityKey{}, identity{WorkID: "work-1"})
	ctx = context.WithValue(ctx, stateKey{}, state)
	if err := g.succeeded(ctx, domain.ToolRunCommand, &entry, report{
		detail: "exit code 7", failed: true, note: &Note{Text: "All tests passed"},
		run: &RunNote{Name: "go test", ExitCode: 7},
	}); err != nil {
		t.Fatal(err)
	}
	if calls.starts != 1 || calls.finishes != 1 || len(calls.recorded) != 0 ||
		calls.finished.Note.Text != "All tests passed" || calls.finished.Note.Run == nil ||
		calls.finished.Note.Run.ExitCode != 7 ||
		calls.finished.Evidence != "ran" || entry.Outcome != OutcomeFailed {
		t.Fatalf("start=%d finish=%d call=%+v outcome=%s", calls.starts, calls.finishes, calls.finished, entry.Outcome)
	}
}

func TestFailedRunningCallUpdateIsReturnedAndRecorded(t *testing.T) {
	calls := &fakeWorkCalls{finishErr: errWorkCallTestFailure}
	activity := &captureRecorder{}
	g := &guard{workCalls: calls, recorder: activity}
	entry := Entry{UserID: "u", Client: "ChatGPT", Project: "p", Summary: "command: go test"}
	state := &callState{taskID: "task", eventID: "event", workID: "work"}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{WorkID: "work"})
	ctx = context.WithValue(ctx, stateKey{}, state)
	err := g.succeeded(ctx, domain.ToolRunCommand, &entry, report{detail: "finished"})
	if err == nil || !strings.Contains(err.Error(), "workspace could not be used") ||
		entry.Outcome != OutcomeFailed ||
		len(activity.entries) != 1 || activity.entries[0].Outcome != OutcomeFailed {
		t.Fatalf("finish error=%v entry=%+v activity=%+v", err, entry, activity.entries)
	}
}

func TestLegacyThreadStillUsesTaskLogWithoutWorkCalls(t *testing.T) {
	legacy := &fakeTaskLog{}
	calls := &fakeWorkCalls{}
	g := &guard{tasks: legacy, workCalls: calls}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{User: "u", Client: "ChatGPT"})
	entry := Entry{UserID: "u", Client: "ChatGPT", Project: "p", Summary: "read file"}
	if err := g.thread(ctx, domain.ToolReadFile, &entry, report{}); err != nil {
		t.Fatal(err)
	}
	if legacy.logs != 1 || calls.attempts != 0 || entry.Task != "legacy-task" {
		t.Fatalf("legacy logs=%d work calls=%d task=%q", legacy.logs, calls.attempts, entry.Task)
	}
}

type fakeWorkCalls struct {
	recordErr error
	finishErr error
	WorkCalls
	finished WorkCall
	recorded []WorkCall
	starts   int
	finishes int
	attempts int
}

func (f *fakeWorkCalls) Start(context.Context, string, string, string, string, WorkCall) (string, string, error) {
	f.starts++
	return "task", "event", nil
}

func (f *fakeWorkCalls) Finish(_ context.Context, _, _, _ string, call WorkCall) error {
	f.finishes++
	f.finished = call
	return f.finishErr
}

func (f *fakeWorkCalls) Record(_ context.Context, _, _, _, _ string, call WorkCall) (string, error) {
	f.attempts++
	f.recorded = append(f.recorded, call)
	return "task", f.recordErr
}

type captureRecorder struct{ entries []Entry }

func (r *captureRecorder) Record(_ context.Context, entry Entry) error {
	r.entries = append(r.entries, entry)
	return nil
}

type fakeTaskLog struct {
	logErr       error
	logs, begins int
}

func (f *fakeTaskLog) Log(context.Context, string, string, string, Note) (string, error) {
	f.logs++
	return "legacy-task", f.logErr
}

func (f *fakeTaskLog) Begin(context.Context, string, string, string, Note) (string, error) {
	f.begins++
	return "legacy-task", nil
}

func (*fakeTaskLog) End(context.Context, string) {}
