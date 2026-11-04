package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// endDelegation closes the job, shows the person how it ended, and answers the delegate call.
func (r *Runner) endDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, d Delegation, status, report string,
) {
	_ = r.deps.Delegations.EndDelegation(ctx, task.ID, d.ID, status, report, r.deps.Now())

	action, text := handoffReported, "The "+d.RoleName+" reported back"
	if status == DelegationFailed {
		action, text = handoffFailed, "The "+d.RoleName+" could not finish"
	}

	plan, detail := planStateAfter(status == DelegationFailed, report)
	state.lock()
	r.trackPlan(ctx, task, d, plan, detail)
	state.unlock()

	r.handoff(ctx, task, d, action, text, report)
	r.result(ctx, task, call.ID, report, status == DelegationFailed)
}

func (r *Runner) handoff(ctx context.Context, task domain.Task, d Delegation, action, text, detail string) {
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventHandoff, Text: text, Technical: detail, Action: action, Agent: d.RoleName, Job: d.ID,
		Model: d.Model, Effort: string(d.Effort), Time: r.deps.Now(),
	})
}

func teammateKeys(roster *Roster) string {
	keys := make([]string, 0, len(roster.Members))
	for _, member := range roster.Members {
		keys = append(keys, member.Key)
	}

	return "Choose one of: " + strings.Join(keys, ", ") + "."
}

func reportText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(The teammate sent no report.)"
	}

	return text
}
