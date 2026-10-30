package bootstrap

import (
	"context"
	"encoding/json"
	"strings"

	molabapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const toolMoLabExecute = "marimo_execute"

func molabExecuteDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name:        toolMoLabExecute,
		Description: "Run Python in a MoLab notebook using the HTTPS URL and temporary token the USER supplied in chat. The app selects both from the same user message automatically; neither belongs in tool arguments. No separate pairing UI or egress approval.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"Python to execute in the notebook"}},"required":["code"],"additionalProperties":false}`),
	}
}

func (a *agentTools) executeMoLab(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if a.molabTasks == nil || scope.TaskID == "" || scope.ProjectID == "" || scope.UserID == "" {
		return problem("MoLab requires an active conversation in an AI Code project.")
	}
	task, err := a.molabTasks.Get(ctx, scope.UserID, scope.TaskID)
	if err != nil || task.ProjectID != scope.ProjectID || task.ArchivedAt != nil ||
		task.Status == taskdomain.StatusDone || task.Status == taskdomain.StatusFailed {
		return problem("The task is closed or is not yours. Start or continue your own conversation.")
	}
	var input struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(call.Input, &input) != nil || strings.TrimSpace(input.Code) == "" || len(input.Code) > 32000 {
		return problem("Provide Python code (at most 32 KB) in marimo_execute.")
	}
	events, err := a.molabTasks.Events(ctx, scope.UserID, scope.TaskID, 0)
	if err != nil {
		return problem("Could not read the current chat request for MoLab.")
	}
	cred, err := molabapp.FromChat(events)
	if err != nil {
		return problem("No valid MoLab URL + token together in a user chat message. Send both in the chat and try again.")
	}
	_, err = a.services(scope.ProjectID)
	if err != nil {
		return problem("The sandbox is unavailable for this project.")
	}
	a.mu.Lock()
	runner := a.runners[scope.ProjectID]
	a.mu.Unlock()
	if runner == nil {
		return problem("The sandbox is unavailable for this project.")
	}
	result := molabapp.Execute(taskContext(ctx, scope), runner, cred.URL, cred.Token, input.Code)
	if result.Problem != "" {
		return problem(result.Problem)
	}
	return taskapp.ToolResult{Output: "MoLab code executed: " + result.Output}
}
