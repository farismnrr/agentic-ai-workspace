package mcp

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	workspacefs "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPWorkEditsAndCommandsStayOnWorkThread(t *testing.T) {
	server, workCalls, legacy, recorder, works, commands := newRecordingMCP(t)
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	scopes := []string{codingdomain.ScopeRead, codingdomain.ScopeWrite, codingdomain.ScopeExec}
	token := &auth.TokenInfo{UserID: "u1", Scopes: scopes, Extra: map[string]any{clientKey: "ChatGPT"}}
	if _, connectErr := server.Connect(t.Context(), verifiedTransport{
		Transport: serverTransport, token: token,
	}, nil); connectErr != nil {
		t.Fatal(connectErr)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	workID := callRecordingWork(t, session)
	assertWorkRecording(t, workCalls, legacy, recorder, works, commands, workID)
}

func newRecordingMCP(t *testing.T) (
	*sdk.Server, *fakeWorkCalls, *fakeTaskLog, *captureRecorder, *fakeWorkSessions, *fixedRunner,
) {
	t.Helper()
	root, err := workspacefs.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	backups, err := backup.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	commands := &fixedRunner{}
	workCalls, legacy, recorder, works := &fakeWorkCalls{}, &fakeTaskLog{}, &captureRecorder{}, &fakeWorkSessions{}
	services := Services{
		ProjectID: "project-1", Editor: codingapp.NewEditor(root, root, backups, opaque.Generator{}),
		Tasks: codingapp.NewTasks(root, commands),
	}
	server := NewServer(recordingWorkspace{services: services}, Options{
		Gate: testGate{allowed: true}, Recorder: recorder, Tasks: legacy,
		Works: works, WorkCalls: workCalls,
	})
	return server, workCalls, legacy, recorder, works, commands
}

func callRecordingWork(t *testing.T, session *sdk.ClientSession) string {
	t.Helper()
	begin, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "begin_work", Arguments: map[string]any{
		"summary": "Make a file", "idempotency_key": "integration-key",
	}})
	requireToolSuccess(t, begin, err)
	workID := begin.StructuredContent.(map[string]any)["work_id"].(string)
	write, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "write_file", Arguments: map[string]any{
		"work_id": workID, "path": "page.txt", "content": "hello",
	}})
	requireToolSuccess(t, write, err)
	command, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "run_command", Arguments: map[string]any{
		"work_id": workID, "command": "check", "timeout_seconds": 10,
	}})
	requireToolSuccess(t, command, err)
	return workID
}

//nolint:gocyclo,cyclop // aggregate assertions cover the separate MCP evidence and ownership invariants.
func assertWorkRecording(t *testing.T, workCalls *fakeWorkCalls, legacy *fakeTaskLog, recorder *captureRecorder,
	works *fakeWorkSessions, commands *fixedRunner, workID string,
) {
	t.Helper()
	if works.state.WorkID != workID {
		t.Fatalf("work session changed: %+v", works.state)
	}
	if len(workCalls.recorded) != 1 || len(workCalls.recorded[0].Files) != 1 ||
		workCalls.recorded[0].Files[0] != "page.txt" || len(workCalls.recorded[0].Undo) == 0 ||
		workCalls.recorded[0].Evidence != "observed" {
		t.Fatalf("file change event = %+v", workCalls.recorded)
	}
	if len(recorder.entries) != 3 || recorder.entries[1].Tool != "write_file" ||
		recorder.entries[1].Undo != nil || works.state.State != "open" {
		t.Fatalf("undo owner, activity rows, or work state changed: activity=%+v work=%+v", recorder.entries, works.state)
	}
	if workCalls.starts != 1 || workCalls.finishes != 1 || workCalls.finished.Phase != "finished" ||
		workCalls.finished.Note.Run == nil || workCalls.finished.Note.Run.ExitCode != 7 || commands.runs != 1 {
		t.Fatalf("command start=%d finish=%d event=%+v runs=%d", workCalls.starts, workCalls.finishes,
			workCalls.finished, commands.runs)
	}
	if legacy.logs != 0 || legacy.begins != 0 {
		t.Fatalf("legacy task was created: %+v", legacy)
	}
}

type recordingWorkspace struct{ services Services }

func (w recordingWorkspace) For(context.Context, string, string, string) (Services, error) {
	return w.services, nil
}

func (recordingWorkspace) Projects(context.Context, string, string) ([]ProjectInfo, error) {
	return nil, nil
}

type fixedRunner struct{ runs int }

func (r *fixedRunner) Run(context.Context, codingdomain.Task) (codingapp.RunResult, error) {
	r.runs++
	return codingapp.RunResult{ExitCode: 7}, nil
}
