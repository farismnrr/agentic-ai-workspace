package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// runUntilTurnLimit keeps the same task and transcript across per-run turn caps.
func (r *Runner) runUntilTurnLimit(ctx context.Context, task domain.Task) outcome {
	var result outcome
	for ctx.Err() == nil {
		result = r.loop(ctx, task)
		if result.stop != stopTurnLimit {
			break
		}
	}
	if result.stop == stopTurnLimit {
		result, _ = r.limitReached(ctx)
	}
	return result
}
