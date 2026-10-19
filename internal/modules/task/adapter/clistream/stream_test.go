package clistream_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/clistream"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func read(t *testing.T, program, output string) ([]application.CLIEvent, application.CLIResult) {
	t.Helper()

	var events []application.CLIEvent

	stream := clistream.New(program, func(event application.CLIEvent) { events = append(events, event) })
	// A tiny write size proves a line split across writes is still read whole.
	for i := 0; i < len(output); i += 7 {
		if _, err := stream.Write([]byte(output[i:min(i+7, len(output))])); err != nil {
			t.Fatal(err)
		}
	}

	return events, stream.Result()
}

func TestClaudeStream(t *testing.T) {
	t.Parallel()

	output := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"s-1"}`,
		`{"type":"stream_event","event":{"delta":{"type":"text_delta","text":"Hel"}}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"echo hi > /workspace/hello.txt","description":"Create hello.txt"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":false}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Write","input":{"file_path":"/workspace/a.txt"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t2","is_error":false}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t3","name":"Read","input":{"file_path":"/workspace/a.txt"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t3","is_error":false}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Done."}]}}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"Done.","session_id":"s-1"}`,
	}, "\n") + "\n"

	events, result := read(t, "claude", output)

	if result.Session != "s-1" || result.Summary != "Done." || result.Reason != "" {
		t.Fatalf("result = %+v", result)
	}

	kinds := make([]application.CLIEventKind, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}

	want := []application.CLIEventKind{application.CLIDelta, application.CLICommand, application.CLIChange, application.CLIText}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v, want %v (a Read must not become a step)", kinds, want)
	}

	if events[1].Text != "Create hello.txt" || events[2].Files[0] != "a.txt" {
		t.Fatalf("events = %+v", events)
	}
}

func TestCodexStream(t *testing.T) {
	t.Parallel()

	output := strings.Join([]string{
		`{"type":"thread.started","thread_id":"th-1"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"i1","type":"command_execution","command":"bash -lc 'go test ./...'","exit_code":1,"status":"failed"}}`,
		`{"type":"item.completed","item":{"id":"i2","type":"command_execution","command":"ls","status":"declined"}}`,
		`{"type":"item.completed","item":{"id":"i3","type":"file_change","changes":[{"path":"/workspace/main.go","kind":"update"}],"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"i4","type":"agent_message","text":"All set."}}`,
		`{"type":"turn.completed","usage":{"input_tokens":1}}`,
	}, "\n") + "\n"

	events, result := read(t, "chatgpt", output)

	if result.Session != "th-1" || result.Summary != "All set." {
		t.Fatalf("result = %+v", result)
	}

	if len(events) != 4 {
		t.Fatalf("events = %+v", events)
	}

	if events[0].ExitCode != 1 || events[0].Text != "Ran go test ./..." || events[1].ExitCode == 0 {
		t.Fatalf("commands = %+v %+v", events[0], events[1])
	}

	if events[2].Files[0] != "main.go" {
		t.Fatalf("change = %+v", events[2])
	}
}

func TestCodexFailure(t *testing.T) {
	t.Parallel()

	_, result := read(t, "chatgpt", `{"type":"turn.failed","error":{"message":"usage limit reached"}}`+"\n")

	if !strings.Contains(result.Reason, "usage limit reached") {
		t.Fatalf("reason = %q", result.Reason)
	}
}
