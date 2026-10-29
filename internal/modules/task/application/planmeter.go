package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// saveMeter tells the task page how full the program's window is. The program keeps its own conversation and
// shortens it itself, so this only shows what it reported; a teammate's turn leaves the lead's meter alone.
func (r *Runner) saveMeter(ctx context.Context, task domain.Task, result CLIResult) {
	if r.role != nil || result.Used <= 0 {
		return
	}

	window := result.Window
	if window <= 0 {
		return // vendor did not report a window: don't manufacture a percentage
	}
	_ = r.deps.Tasks.SetContext(ctx, task.ID, min(result.Used, window), window)
}
