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

func TestCodexStartedToolPreventsUnsafeAutomaticReplay(t *testing.T) {
	var started int
	stream := clistream.New("chatgpt", func(event application.CLIEvent) {
		if event.Kind == application.CLIActivity {
			started++
		}
	})
	_, err := io.WriteString(stream, "{\"type\":\"item.started\",\"item\":{\"type\":\"command_execution\",\"command\":\"touch file\"}}\n")
	if err != nil || started != 1 {
		t.Fatalf("command start not recorded: %d %v", started, err)
	}
}
func TestClaudeToolUsePreventsUnsafeAutomaticReplay(t *testing.T) {
	var started int
	stream := clistream.New("claude", func(event application.CLIEvent) {
		if event.Kind == application.CLIActivity {
			started++
		}
	})
	_, err := io.WriteString(stream, "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"tool_use\",\"id\":\"u1\",\"name\":\"Bash\",\"input\":{\"command\":\"touch file\"}}]}}\n")
	if err != nil || started != 1 {
		t.Fatalf("tool use start not recorded: %d %v", started, err)
	}
}
func TestClaudeUsesLastModelsWindowInsteadOfLargestModel(t *testing.T) {
	stream := clistream.New("claude", func(application.CLIEvent) {})
	_, err := io.WriteString(stream, "{\"type\":\"assistant\",\"message\":{\"model\":\"small\",\"usage\":{\"input_tokens\":24000},\"content\":[]}}\n{\"type\":\"result\",\"modelUsage\":{\"small\":{\"contextWindow\":32000},\"large\":{\"contextWindow\":200000}},\"result\":\"done\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	got := stream.Result()
	if got.Window != 32000 || got.Used != 24000 {
		t.Fatalf("wrong model window: %+v", got)
	}
}
func TestClaudeMultipleUnknownWindowsRemainUnknown(t *testing.T) {
	stream := clistream.New("claude", func(application.CLIEvent) {})
	_, err := io.WriteString(stream, "{\"type\":\"result\",\"modelUsage\":{\"a\":{\"contextWindow\":32000},\"b\":{\"contextWindow\":200000}},\"result\":\"done\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Result().Window != 0 {
		t.Fatal("guessed window of an unidentified model")
	}
}

func TestClaudeStartedToolStreamingPreventsReplay(t *testing.T) {
	var starts int
	stream := clistream.New("claude", func(event application.CLIEvent) {
		if event.Kind == application.CLIActivity {
			starts++
		}
	})
	_, err := io.WriteString(stream, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","id":"toolu-1","name":"Bash"}}}`+"\n")
	if err != nil || starts != 1 {
		t.Fatalf("streaming tool start not tracked: %d, %v", starts, err)
	}
}
