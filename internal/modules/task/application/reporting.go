package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// report puts what a tool did into the transcript and the thread.
func (r *Runner) report(ctx context.Context, task domain.Task, call Call, described Described, result ToolResult) {
	if shot := r.saveShot(ctx, task, call, result); shot != "" {
		_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
			Kind: domain.EventShot, Text: described.Text, Image: shot, Time: r.deps.Now(),
		})

		return
	}

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
	r.logStep(ctx, task, call, event, result.IsError)
	r.announceFiles(task, call)

	if result.Link != "" && !result.IsError {
		_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
			Kind: domain.EventSite, Text: "Your app is running", URL: result.Link, Time: r.deps.Now(),
		})
	}
}

// logStep tells the activity log about a step that changed or ran something; looking around is not logged.
func (r *Runner) logStep(ctx context.Context, task domain.Task, call Call, event domain.Event, failed bool) {
	if r.deps.Activity == nil || r.deps.Tools.Risk(call.Name) == domain.RiskNone {
		return
	}

	r.deps.Activity.Step(ctx, Step{
		UserID: task.UserID, ProjectID: task.ProjectID, TaskID: task.ID, Tool: call.Name,
		Summary: event.Text, Detail: event.Technical, Failed: failed,
	})
}

// announceFiles tells anyone looking at the project's files that a step which may have changed them has run: an
// edit, a command or a check. Looking around changes nothing.
func (r *Runner) announceFiles(task domain.Task, call Call) {
	if r.deps.Notify != nil && r.deps.Tools.Risk(call.Name) != domain.RiskNone {
		r.deps.Notify.FilesChanged(task.ProjectID)
	}
}
