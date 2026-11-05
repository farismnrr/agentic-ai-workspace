package application

import (
	"context"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// endDelegation closes the job, shows the person how it ended, and answers the delegate call.
func (r *Runner) endDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, d Delegation, status, report string,
) {
	actualStatus, actualReport, err := r.deps.Delegations.EndDelegation(
		ctx, task.ID, d.ID, status, report, r.deps.Now(),
	)
	if err != nil {
		if ctx.Err() != nil {
			r.settledAfterCancellation(ctx, task, call, d)
			return
		}
		r.result(ctx, task, call.ID, "I could not safely save this teammate's result. Check the task before retrying.", true)
		return
	}
	if actualStatus != status {
		if actualStatus == DelegationCancelled {
			r.result(ctx, task, call.ID, "This teammate's job was cancelled before its report was saved.", true)
		} else {
			r.result(ctx, task, call.ID, actualReport, actualStatus != DelegationDone)
		}
		return
	}
	report = actualReport

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

func (r *Runner) settledAfterCancellation(ctx context.Context, task domain.Task, call Call, d Delegation) {
	readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	current, found, err := r.deps.Delegations.FindDelegation(readCtx, task.ID, d.ID)
	if err != nil || !found {
		r.result(ctx, task, call.ID, "I could not safely confirm this teammate's final status.", true)
		return
	}
	if current.Status == DelegationCancelled {
		r.result(ctx, task, call.ID, "This teammate's job was cancelled before its report was saved.", true)
		return
	}
	if current.Status == DelegationDone || current.Status == DelegationFailed {
		r.result(ctx, task, call.ID, current.Report, current.Status != DelegationDone)
		return
	}
	r.result(ctx, task, call.ID, "I could not safely confirm this teammate's final status.", true)
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
