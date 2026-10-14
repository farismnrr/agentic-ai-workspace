package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// RunCommand runs one shell command line in the workspace's sandbox.
func (t *Tasks) RunCommand(ctx context.Context, command string, timeoutSeconds int) (RunResult, error) {
	task, err := domain.CommandTask(command, timeoutSeconds)
	if err != nil {
		return RunResult{}, fmt.Errorf("run command: %w", err)
	}

	result, err := t.runner.Run(ctx, task)
	if err != nil {
		return RunResult{}, fmt.Errorf("run command: %w", err)
	}

	return result, nil
}
