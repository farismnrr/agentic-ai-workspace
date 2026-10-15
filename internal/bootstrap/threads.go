package bootstrap

import (
	"context"
	"fmt"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxStepText = 200

// agentSteps lets the agent write what it did into the activity log, naming the project and task.
type agentSteps struct {
	log *spaceapp.ActivityLog
}

func (a agentSteps) Step(ctx context.Context, step taskapp.Step) {
	outcome := spacedomain.OutcomeOK
	if step.Failed {
		outcome = spacedomain.OutcomeFailed
	}

	// A log failure must never stop the agent. There is no undo here: the task's own thread undoes its changes.
	_, _ = a.log.Record(ctx, spacedomain.Activity{
		UserID: step.UserID, Space: "coding", Tool: step.Tool, Summary: cut(step.Summary), Detail: cut(step.Detail),
		Outcome: outcome, ProjectID: step.ProjectID, TaskID: step.TaskID,
	})
}

// appThreads lets a connected app's conversation be read as a task, next to the ones the agent ran.
type appThreads struct {
	tasks *taskapp.Service
}

func (a appThreads) Log(ctx context.Context, userID, client, projectID string, note codingmcp.Note) (string, error) {
	if client == "" {
		client = unnamedApp
	}

	id, err := a.tasks.Record(ctx, userID, taskapp.RecordInput{
		ProjectID: projectID, Source: client, Event: eventOf(note),
	})
	if err != nil {
		return "", fmt.Errorf("record the app's step: %w", err)
	}

	return id, nil
}

func eventOf(note codingmcp.Note) taskdomain.Event {
	event := taskdomain.Event{Kind: taskdomain.EventAction, Text: note.Text, Technical: note.Technical}

	switch note.Kind {
	case codingmcp.NoteUser:
		event.Kind = taskdomain.EventRequest
	case codingmcp.NoteAssistant:
		event.Kind = taskdomain.EventMessage
	case codingmcp.NoteAction:
		if note.Run != nil {
			event.Run = &taskdomain.RunInfo{Name: note.Run.Name, ExitCode: note.Run.ExitCode, TimedOut: note.Run.TimedOut}
		}
	}

	return event
}

func cut(text string) string {
	runes := []rune(text)
	if len(runes) <= maxStepText {
		return text
	}

	return string(runes[:maxStepText]) + "…"
}
