package clistream_test

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/clistream"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"io"
	"testing"
)

func TestCodexCumulativeUsageIsNotDisplayedAsContext(t *testing.T) {
	stream := clistream.New("chatgpt", func(application.CLIEvent) {})
	_, err := io.WriteString(stream, "{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":350000}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := stream.Result().Used; got != 0 {
		t.Fatalf("cumulative usage must not be shown as active context: %d", got)
	}
}
func TestCodexLastCallUsageCanBeDisplayed(t *testing.T) {
	stream := clistream.New("chatgpt", func(application.CLIEvent) {})
	_, err := io.WriteString(stream, "{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":350000},\"last_token_usage\":{\"input_tokens\":24000}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := stream.Result().Used; got != 24000 {
		t.Fatalf("last call usage = %d, want 24000", got)
	}
}

func TestCodexTokenCountEmitsLiveContext(t *testing.T) {
	var got application.CLIEvent
	stream := clistream.New("chatgpt", func(e application.CLIEvent) {
		if e.Kind == application.CLIUsage {
			got = e
		}
	})
	_, err := io.WriteString(stream, "{\"type\":\"thread.token_count\",\"token_info\":{\"total_token_usage\":{\"input_tokens\":500000},\"last_token_usage\":{\"input_tokens\":23000}}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Used != 23000 {
		t.Fatalf("live last-call usage %d, want 23000", got.Used)
	}
}

func TestClaudeEmitsLiveContextFromAssistantUsage(t *testing.T) {
	var got application.CLIEvent
	stream := clistream.New("claude", func(e application.CLIEvent) {
		if e.Kind == application.CLIUsage {
			got = e
		}
	})
	_, err := io.WriteString(stream, "{\"type\":\"assistant\",\"message\":{\"usage\":{\"input_tokens\":100,\"cache_read_input_tokens\":2000,\"cache_creation_input_tokens\":500,\"output_tokens\":200},\"content\":[]}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Used != 2800 {
		t.Fatalf("live context usage %d, want 2800", got.Used)
	}
}
