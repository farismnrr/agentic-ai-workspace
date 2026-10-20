package clistream_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/clistream"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func feed(program, output string) application.CLIResult {
	stream := clistream.New(program, func(application.CLIEvent) {})
	_, _ = stream.Write([]byte(output))

	return stream.Result()
}

func TestClaudeReportsTheSizeOfItsLastCallAndItsWindow(t *testing.T) {
	t.Parallel()

	got := feed("claude", `{"type":"assistant","message":{"usage":{"input_tokens":10,"cache_read_input_tokens":1000,`+
		`"cache_creation_input_tokens":90,"output_tokens":5},"content":[]}}
{"type":"assistant","message":{"usage":{"input_tokens":20,"cache_read_input_tokens":2000,`+
		`"cache_creation_input_tokens":0,"output_tokens":7},"content":[]}}
{"type":"result","session_id":"s","result":"ok","modelUsage":{"claude-x":{"contextWindow":1000000}}}
`)

	if got.Used != 2027 || got.Window != 1_000_000 {
		t.Fatalf("used %d of %d, want 2027 of 1000000", got.Used, got.Window)
	}
}

func TestCodexReportsWhatTheTurnCarried(t *testing.T) {
	t.Parallel()

	got := feed("chatgpt", `{"type":"turn.completed","usage":{"input_tokens":4321,"output_tokens":9}}
`)

	if got.Used != 4321 || got.Window != 0 {
		t.Fatalf("used %d of %d, want 4321 of nothing", got.Used, got.Window)
	}
}
