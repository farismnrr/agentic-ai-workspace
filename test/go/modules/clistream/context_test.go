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
