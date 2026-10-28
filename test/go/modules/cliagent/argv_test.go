package cliagent_test

import (
	"slices"
	"strings"
	"testing"

	cliagentcommand "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/command"
	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestProgramArgvModelAndEffort(t *testing.T) {
	t.Parallel()

	codex := []cliagentdomain.Model{{ID: "gpt-x", Efforts: []string{"low", "ultra"}}}
	cases := map[string]struct {
		models []cliagentdomain.Model
		req    taskapp.CLIRequest
	}{
		"--model opus --effort high --resume s1": {
			cliagentdomain.ClaudeModels(), taskapp.CLIRequest{Program: "claude", Model: "opus", Effort: "high", Session: "s1"},
		},
		`-m gpt-x -c model_reasoning_effort="ultra" -`: {
			codex, taskapp.CLIRequest{Program: "chatgpt", Model: "gpt-x", Effort: "ultra"},
		},
		`-c model_reasoning_effort="low" -`: {
			cliagentdomain.OwnChoice(), taskapp.CLIRequest{Program: "chatgpt", Model: "default", Effort: "low"},
		},
	}

	for want, c := range cases {
		if got := strings.Join(cliagentcommand.ProgramArgv(c.req, c.models, 0), " "); !strings.Contains(got, want) {
			t.Errorf("argv %q lacks %q", got, want)
		}
	}
}

func TestProgramArgvIgnoresUnknownChoices(t *testing.T) {
	t.Parallel()

	req := taskapp.CLIRequest{Program: "claude", Model: "--bad", Effort: "turbo"}

	argv := cliagentcommand.ProgramArgv(req, cliagentdomain.ClaudeModels(), 0)
	for _, banned := range []string{"--model", "--effort", "--bad", "turbo"} {
		if slices.Contains(argv, banned) {
			t.Errorf("an unknown choice reached the program: %v", argv)
		}
	}
}
