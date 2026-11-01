package bootstrap

import (
	"fmt"
	"strings"

	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxCommandName = 60

// commandName is how a command is listed among the checks that ran: its first line, kept short.
func commandName(command string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(command), "\n")
	runes := []rune(line)

	if len(runes) > maxCommandName {
		return string(runes[:maxCommandName]) + "…"
	}

	return line
}

// ran words a finished run for the model and records it as a check that really ran.
func ran(name, label, command string, result codingapp.RunResult, err error) taskapp.ToolResult {
	if err != nil {
		return failure(err)
	}

	verdict := fmt.Sprintf("exit code %d", result.ExitCode)

	switch {
	case result.TimedOut:
		verdict = "timed out"
	case result.ExitCode == 0:
		verdict = "passed (exit code 0)"
	}

	return taskapp.ToolResult{
		Output: fmt.Sprintf("%s finished: %s.\n%s", label, verdict, result.Output),
		Note:   verdict,
		Run: &taskdomain.RunInfo{
			Name: name, Command: command, EffectPaths: result.EffectPaths, EffectsKnown: result.EffectsKnown,
			ExitCode: result.ExitCode, TimedOut: result.TimedOut,
			TestExecuted: goTestsExecuted(command, result.Output, result.Truncated),
		},
	}
}
