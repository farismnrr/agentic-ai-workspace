package clistream_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/clistream"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// read feeds a recorded output to the parser in small writes, which proves a line split across writes is read whole.
func read(t *testing.T, program, file string) ([]application.CLIEvent, application.CLIResult) {
	t.Helper()

	root, err := os.OpenRoot("testdata")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = root.Close() }()

	raw, err := root.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	var events []application.CLIEvent

	stream := clistream.New(program, func(event application.CLIEvent) { events = append(events, event) })
	for i := 0; i < len(raw); i += 7 {
		_, _ = stream.Write(raw[i:min(i+7, len(raw))])
	}

	return events, stream.Result()
}

// outline writes a turn as one line per step, so a test compares it to what a person would see.
func outline(events []application.CLIEvent, result application.CLIResult) string {
	lines := []string{"session=" + result.Session, "summary=" + result.Summary, "reason=" + result.Reason}

	for _, event := range events {
		files := strings.Join(event.Files, ",")
		lines = append(lines, fmt.Sprintf("%d|%s|%s|%d", event.Kind, event.Text, files, event.ExitCode))
	}

	return strings.Join(lines, "\n")
}

func TestStreams(t *testing.T) {
	t.Parallel()

	// The Read in the Claude output is not a step; a declined Codex command counts as failed.
	cases := map[string][]string{
		"claude.jsonl": {
			"session=s-1", "summary=Done.", "reason=", "2|Hel||0", "4|Create hello.txt||0", "3|Wrote a.txt|a.txt|0",
			"1|Done.||0",
		},
		"codex.jsonl": {
			"session=th-1", "summary=All set.", "reason=", "4|Ran go test ./...||1", "4|Ran ls||1",
			"3|Changed main.go|main.go|0", "1|All set.||0",
		},
	}

	for file, want := range cases {
		program := "claude"
		if strings.HasPrefix(file, "codex") {
			program = "chatgpt"
		}

		events, result := read(t, program, file)
		if got := outline(events, result); got != strings.Join(want, "\n") {
			t.Errorf("%s:\n%s\nwant:\n%s", file, got, strings.Join(want, "\n"))
		}
	}
}

func TestCodexFailure(t *testing.T) {
	t.Parallel()

	stream := clistream.New("chatgpt", func(application.CLIEvent) {})
	_, _ = stream.Write([]byte(`{"type":"turn.failed","error":{"message":"usage limit reached"}}` + "\n"))

	if got := stream.Result().Reason; !strings.Contains(got, "usage limit reached") {
		t.Fatalf("reason = %q", got)
	}
}
