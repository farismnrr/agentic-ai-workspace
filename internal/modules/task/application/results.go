package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (r *Runner) result(ctx context.Context, task domain.Task, callID, text string, isError bool) {
	if callID == "" {
		_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{Kind: domain.BlockText, Text: text})
		return
	}
	r.resultUpTo(ctx, task, callID, text, isError, r.lim.MaxResult)
}

// resultUpTo answers a call with a text cut at its own limit, for the few results that are meant to be read whole.
func (r *Runner) resultUpTo(ctx context.Context, task domain.Task, callID, text string, isError bool, limit int) {
	_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{
		Kind: domain.BlockToolResult, ID: callID, Text: truncate(text, limit), IsError: isError,
	})
}

func (r *Runner) action(ctx context.Context, task domain.Task, event domain.Event) {
	event.Kind, event.Time = domain.EventAction, r.deps.Now()
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, event)
}

// resultImage answers a call with a text and the screenshot it carries (saved under image).
func (r *Runner) resultImage(ctx context.Context, task domain.Task, callID, text, image string) {
	_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{
		Kind: domain.BlockToolResult, ID: callID, Text: truncate(text, r.lim.MaxResult), Image: image,
	})
}
