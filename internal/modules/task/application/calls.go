package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// process works through the tool calls that still need a result, in order. It
// stops at the first call that has to wait for the person, or that ends the task.
func (r *Runner) process(ctx context.Context, task domain.Task, pending []Call) (outcome, bool) {
	for index, call := range pending {
		result, done := r.handle(ctx, task, call)
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

func (r *Runner) handle(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	switch call.Name {
	case toolUpdatePlan:
		r.updatePlan(ctx, task, call)

		return outcome{}, false
	case toolAskUser:
		return r.askUser(ctx, task, call)
	case toolFinish:
		return r.finish(ctx, task, call), true
	}

	return r.tool(ctx, task, call)
}

// tool runs a project tool, asking the person first when the step is risky enough.
func (r *Runner) tool(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	described := r.deps.Tools.Describe(call)

	if risk := r.deps.Tools.Risk(call.Name); needsApproval(risk) {
		decision, wait := r.approval(ctx, task, call, risk, described)
		if wait {
			return outcome{stop: stopWaiting}, true
		}

		if decision == domain.Denied {
			r.result(ctx, task, call.ID, "The person did not allow this step. Try another way, or ask them.", true)
			r.action(ctx, task, domain.Event{Text: "Skipped, because you said no: " + lowerFirst(described.Text)})

			return outcome{}, false
		}
	}

	result := r.deps.Tools.Execute(ctx, task.ProjectID, call)
	r.result(ctx, task, call.ID, result.Output, result.IsError)

	event := domain.Event{
		Text: described.Text, Technical: described.Technical, Files: result.Files, Undo: result.Undo, Run: result.Run,
	}
	if result.Note != "" {
		event.Technical += " → " + result.Note
	}

	if result.IsError {
		event.Text = "Tried to, but it did not work: " + lowerFirst(described.Text)
		event.Technical = described.Technical + " → " + truncate(result.Output, maxModelError)
	}

	r.action(ctx, task, event)

	return outcome{}, false
}

// needsApproval is the approval policy: looking around (read tools) runs on its own,
// anything that changes files or runs a command asks the person first.
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

func (r *Runner) result(ctx context.Context, task domain.Task, callID, text string, isError bool) {
	_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{
		Kind: domain.BlockToolResult, ID: callID, Text: truncate(text, r.lim.MaxResult), IsError: isError,
	})
}

func (r *Runner) action(ctx context.Context, task domain.Task, event domain.Event) {
	event.Kind, event.Time = domain.EventAction, r.deps.Now()
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, event)
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
