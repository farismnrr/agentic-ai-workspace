package bootstrap

import (
	"context"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// appWorkCalls adapts MCP activity to the task-owned work event use cases.
type appWorkCalls struct{ works *taskapp.Works }

func (a appWorkCalls) Start(
	ctx context.Context, user, client, project, workID string, call codingmcp.WorkCall,
) (string, string, error) {
	taskID, eventID, err := a.works.StartCall(ctx, user, client, project, workID, workEvent(call))
	return taskID, eventID, workBridgeError("start connected-app call", err)
}

func (a appWorkCalls) Finish(
	ctx context.Context, taskID, workID, eventID string, call codingmcp.WorkCall,
) error {
	return workBridgeError("finish connected-app call", a.works.FinishCall(
		ctx, taskID, workID, eventID, workEvent(call),
	))
}

func (a appWorkCalls) Record(
	ctx context.Context, user, client, project, workID string, call codingmcp.WorkCall,
) (string, error) {
	taskID, err := a.works.RecordCall(ctx, user, client, project, workID, workEvent(call))
	return taskID, workBridgeError("record connected-app call", err)
}

func workEvent(call codingmcp.WorkCall) taskdomain.Event {
	event := taskdomain.Event{
		Kind: taskdomain.EventAction, Text: call.Note.Text, Technical: call.Note.Technical,
		Files: call.Files, Undo: call.Undo,
		Work: &taskdomain.WorkRef{Evidence: taskdomain.EvidenceKind(call.Evidence), Phase: call.Phase},
	}
	if call.Note.Run != nil {
		event.Run = &taskdomain.RunInfo{
			Name: call.Note.Run.Name, ExitCode: call.Note.Run.ExitCode, TimedOut: call.Note.Run.TimedOut,
		}
	}
	return event
}
