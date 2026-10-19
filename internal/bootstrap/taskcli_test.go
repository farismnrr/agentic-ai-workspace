package bootstrap

import (
	"slices"
	"strings"
	"testing"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestProgramArgvModelAndEffort(t *testing.T) {
	t.Parallel()

	cases := map[string]taskapp.CLIRequest{
		"--model opus --effort high --resume s1": {Program: "claude", Model: "opus", Effort: "high", Session: "s1"},
		`-c model_reasoning_effort="low" -`:      {Program: "chatgpt", Model: "default", Effort: "low"},
	}

	for want, req := range cases {
		if got := strings.Join(programArgv(req), " "); !strings.Contains(got, want) {
			t.Errorf("argv %q lacks %q", got, want)
		}
	}
}

func TestProgramArgvIgnoresUnknownChoices(t *testing.T) {
	t.Parallel()

	argv := programArgv(taskapp.CLIRequest{Program: "claude", Model: "--bad", Effort: "turbo"})
	for _, banned := range []string{"--model", "--effort", "--bad", "turbo"} {
		if slices.Contains(argv, banned) {
			t.Errorf("an unknown choice reached the program: %v", argv)
		}
	}
}
