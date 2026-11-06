package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// needsApproval is the approval policy: looking around (read tools) runs on its own,
// anything that changes files or runs a command asks the person first, unless they skip approvals.
func needsApproval(risk domain.Risk) bool {
	return risk != domain.RiskNone
}

// approval looks up (or creates) the approval event for a tool call. wait is
// true while the person has not answered.
func (r *Runner) approval(
	ctx context.Context, task domain.Task, call Call, risk domain.Risk, described Described,
) (decision string, wait bool) {
	event, found, err := r.deps.Events.EventByToolUse(ctx, task.ID, call.ID)
	if err != nil {
		return "", true
	}

	if !found {
		_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
			Kind: domain.EventApproval, Text: described.Ask, Action: described.Technical, Risk: risk,
			ToolUseID: call.ID, Time: r.deps.Now(),
		})
		_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusNeedsInput, r.deps.Now())

		return "", true
	}

	return event.Resolved, event.Resolved == ""
}
