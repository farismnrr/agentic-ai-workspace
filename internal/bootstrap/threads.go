package bootstrap

import (
	"context"
	"fmt"
	"strings"

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
	id, err := a.tasks.Record(ctx, userID, appStep(client, projectID, note))
	if err != nil {
		return "", fmt.Errorf("record the app's step: %w", err)
	}

	return id, nil
}

func (a appThreads) Begin(ctx context.Context, userID, client, projectID string, note codingmcp.Note) (string, error) {
	id, err := a.tasks.Begin(ctx, userID, appStep(client, projectID, note))
	if err != nil {
		return "", fmt.Errorf("record the app's step: %w", err)
	}

	return id, nil
}

func (a appThreads) End(ctx context.Context, taskID string) { a.tasks.End(ctx, taskID) }

func appStep(client, projectID string, note codingmcp.Note) taskapp.RecordInput {
	return taskapp.RecordInput{ProjectID: projectID, Source: appName(client), Event: eventOf(note)}
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
			// These facts come from the server's completed runner result,
			// never from the assistant's claims or the tool response body.
			command := recordedWorkCommand(note.Run)
			event.Run = &taskdomain.RunInfo{
				Name: note.Run.Name, Command: command,
				EffectPaths:  append([]string(nil), note.Run.EffectPaths...),
				EffectsKnown: note.Run.EffectsKnown, ExitCode: note.Run.ExitCode,
				TimedOut:     note.Run.TimedOut,
				TestExecuted: goTestsExecuted(command, note.Run.Output, note.Run.Truncated),
			}
		}
	}

	return event
}

// Shell commands normally stay labelled "bash -c ..." and cannot count as
// verification. Promote only a literal, direct Go test: the owner of the
// command runner supplied this exact executed script, and the shared task
// validator still checks for real non-no-op test output.
func recordedWorkCommand(run *codingmcp.RunNote) string {
	raw := strings.TrimSpace(run.ShellCommand)
	if raw != "" && !strings.ContainsAny(raw, ";|&><`$(){}\\\r\n#'\"") &&
		taskapp.GoTestExecutionEvidenceAllowed(raw) {
		return raw
	}
	return run.Command
}

func cut(text string) string {
	runes := []rune(text)
	if len(runes) <= maxStepText {
		return text
	}

	return string(runes[:maxStepText]) + "…"
}
