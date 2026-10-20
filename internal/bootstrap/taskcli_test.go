package bootstrap

import (
	"slices"
	"strings"
	"testing"

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
		if got := strings.Join(programArgv(c.req, c.models, 0), " "); !strings.Contains(got, want) {
			t.Errorf("argv %q lacks %q", got, want)
		}
	}
}

func TestProgramArgvIgnoresUnknownChoices(t *testing.T) {
	t.Parallel()

	req := taskapp.CLIRequest{Program: "claude", Model: "--bad", Effort: "turbo"}

	argv := programArgv(req, cliagentdomain.ClaudeModels(), 0)
	for _, banned := range []string{"--model", "--effort", "--bad", "turbo"} {
		if slices.Contains(argv, banned) {
			t.Errorf("an unknown choice reached the program: %v", argv)
		}
	}
}

func TestAChosenWindowReachesTheProgram(t *testing.T) {
	t.Parallel()

	codex := programArgv(taskapp.CLIRequest{Program: "chatgpt"}, cliagentdomain.OwnChoice(), 150_000)
	if !strings.Contains(strings.Join(codex, " "), "-c model_context_window=150000") {
		t.Errorf("codex argv lacks the window: %v", codex)
	}

	if env := windowEnv("claude", 150_000); len(env) != 1 || env[0] != "CLAUDE_CODE_AUTO_COMPACT_WINDOW=150000" {
		t.Errorf("claude env = %v", env)
	}

	if windowEnv("claude", 0) != nil || windowEnv("chatgpt", 150_000) != nil {
		t.Error("no window chosen, or a program that takes it as an option, must add no variable")
	}
}
