package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// process works through the tool calls that still need a result, in order. It
// stops at the first call that has to wait for the person, or that ends the task.
func (r *Runner) process(
	ctx context.Context, task domain.Task, state *runState, pending []Call,
) (outcome, bool) {
	for index, call := range pending {
		result, done := r.handle(ctx, task, state, call)
		if !done {
			continue
		}

		if result.stop == stopFinished {
			r.skip(ctx, task, pending[index+1:])
		}

		return result, true
	}

	return outcome{}, false
}

func (r *Runner) handle(ctx context.Context, task domain.Task, state *runState, call Call) (outcome, bool) {
	if why := r.refusal(call.Name); why != "" {
		r.result(ctx, task, call.ID, why, true)

		return outcome{}, false
	}

	switch call.Name {
	case toolUpdatePlan:
		r.updatePlan(ctx, task, call)

		return outcome{}, false
	case toolAskUser:
		return r.askUser(ctx, task, call)
	case toolFinish:
		return r.tryFinish(ctx, task, state, call)
	case toolDelegate:
		return r.delegate(ctx, task, state, call)
	case toolReport:
		return r.reportBack(ctx, task, call)
	case toolUseSkill:
		r.useSkill(ctx, task, call)

		return outcome{}, false
	case toolReadReference:
		r.readReference(ctx, task, call)

		return outcome{}, false
	}

	return r.tool(ctx, task, call)
}

// tool runs a project tool, unless the person switched it off or has to say yes first.
func (r *Runner) tool(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	described := r.deps.Tools.Describe(call)

	if blocked, out, done := r.checkTool(ctx, task, call, described); blocked {
		return out, done
	}

	// A small task has no plan to move it out of "planning", so the first step that changes or runs
	// something does.
	if len(task.Plan) == 0 && r.deps.Tools.Risk(call.Name) != domain.RiskNone {
		_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, r.deps.Now())
	}

	r.report(ctx, task, call, described, r.deps.Tools.Execute(ctx, task.ProjectID, call))

	return outcome{}, false
}

// checkTool decides whether a call may run now. blocked is true when it must not: either the
// run goes on without it, or done says the task has to wait for the person's answer.
func (r *Runner) checkTool(
	ctx context.Context, task domain.Task, call Call, described Described,
) (blocked bool, out outcome, done bool) {
	if !r.toolOn(ctx, task.UserID, call.Name) {
		r.result(ctx, task, call.ID, toolOffText, true)
		r.action(ctx, task, domain.Event{Text: "Skipped, because that tool is turned off: " + lowerFirst(described.Text)})

		return true, outcome{}, false
	}

	risk := r.deps.Tools.Risk(call.Name)
	if !needsApproval(risk) || r.skipsApprovals(ctx, task.UserID) {
		return false, outcome{}, false
	}

	decision, wait := r.approval(ctx, task, call, risk, described)
	if wait {
		return true, outcome{stop: stopWaiting}, true
	}

	if decision == domain.Denied {
		r.result(ctx, task, call.ID, "The person did not allow this step. Try another way, or ask them.", true)
		r.action(ctx, task, domain.Event{Text: "Skipped, because you said no: " + lowerFirst(described.Text)})

		return true, outcome{}, false
	}

	return false, outcome{}, false
}

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

func lowerFirst(text string) string {
	if text == "" {
		return text
	}

	runes := []rune(text)
	if runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] += 'a' - 'A'
	}

	return string(runes)
}
