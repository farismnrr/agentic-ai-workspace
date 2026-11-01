package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// settle records how a run ended.
func (r *Runner) settle(ctx context.Context, task domain.Task, result outcome) {
	if result.stop != stopFailed {
		return
	}

	// Manual Stop may have completed while a cancelled model call was unwinding.
	// Make the check and status change atomic, otherwise a late worker can
	// overwrite Done with Failed and put the issue back in Needs Attention.
	changed, err := r.deps.Tasks.FailUnlessDone(ctx, task.ID, r.deps.Now())
	if err != nil || !changed {
		return
	}
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventError, Text: result.text, Hint: result.hint, Time: r.deps.Now(),
	})
}
