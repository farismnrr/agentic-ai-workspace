package bootstrap

import (
	"testing"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
)

func TestExternalWorkPreservesServerRecordedCheckEvidence(t *testing.T) {
	const command = "go test -count=1 -run TestExternalDeepReview ./test/go/app/connectors"
	event := eventOf(codingmcp.Note{
		Kind: codingmcp.NoteAction,
		Run: &codingmcp.RunNote{
			Name: "command", Command: "bash -c " + command, ShellCommand: command,
			Output:       "ok  github.com/farismnrr/agentic-ai-workspace/test/go/app/connectors 0.344s\n",
			EffectsKnown: true, ExitCode: 0,
		},
	})
	if event.Run == nil || event.Run.Command != command ||
		!event.Run.EffectsKnown || !event.Run.TestExecuted || event.Run.ExitCode != 0 {
		t.Fatalf("genuine test evidence lost in MCP work bridge: %+v", event.Run)
	}
}

func TestExternalWorkRejectsShellAndFakeTestEvidence(t *testing.T) {
	cases := []struct {
		name, command, output string
		truncated             bool
	}{
		{"shell injection", "go test -count=1 ./...; echo bogus", "ok  pkg 0.200s", false},
		{"shell newline", "go test -count=1 ./...\necho bogus", "ok  pkg 0.200s", false},
		{"shell command substitution", "go test -count=1 ./... $(echo bogus)", "ok  pkg 0.200s", false},
		{"list only", "go test -list . ./...", "ok  pkg 0.200s", false},
		{"no tests ran", "go test -count=1 ./...", "ok  pkg 0.200s [no tests to run]", false},
		{"truncated result", "go test -count=1 ./...", "ok  pkg 0.200s", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note := codingmcp.Note{Kind: codingmcp.NoteAction, Run: &codingmcp.RunNote{
				Name: "command", Command: "bash -c " + tc.command,
				ShellCommand: tc.command, Output: tc.output, Truncated: tc.truncated,
				EffectsKnown: true,
			}}
			got := eventOf(note)
			if got.Run == nil || got.Run.TestExecuted {
				t.Fatalf("unsafe or unverified command counted as real executed test: %+v", got.Run)
			}
			if got.Run.Command == tc.command && (tc.name == "shell injection" ||
				tc.name == "shell newline" || tc.name == "shell command substitution") {
				t.Fatalf("shell syntax was promoted to a trusted command: %+v", got.Run)
			}
		})
	}
}

func TestExternalWorkRetainsFailureAndChangedPaths(t *testing.T) {
	note := codingmcp.Note{Kind: codingmcp.NoteAction, Run: &codingmcp.RunNote{
		Name: "verify", Command: "go test -count=1 ./...", ExitCode: 1,
		EffectsKnown: true, EffectPaths: []string{"internal/example.go"},
		Output: "FAIL\n",
	}}
	got := eventOf(note)
	if got.Run == nil || got.Run.ExitCode != 1 || got.Run.TestExecuted ||
		len(got.Run.EffectPaths) != 1 || got.Run.EffectPaths[0] != "internal/example.go" {
		t.Fatalf("failure and file effects were lost: %+v", got.Run)
	}
}
