package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const toolMoLabExecute = "marimo_execute"
const inspectionCode = "import marimo._code_mode as cm; help(cm)"

func molabExecuteDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name:        toolMoLabExecute,
		Description: "Execute Python in the notebook paired to this task using its temporary secret. First call inspects marimo._code_mode. Never ask for or include token or URL in tool arguments. Requires approved task egress.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"Python to execute in the active MoLab notebook; code must not contain a token"}},"required":["code"],"additionalProperties":false}`),
	}
}

func (a *agentTools) executeMoLab(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if a.molab == nil || a.molabTasks == nil || scope.TaskID == "" || scope.ProjectID == "" || scope.UserID == "" {
		return problem("A task-scoped MoLab session is not available.")
	}
	if a.egress == nil {
		return problem("Per-task network grants are disabled. An administrator must verify and enable EGRESS_GRANTS before MoLab access.")
	}
	task, err := a.molabTasks.Get(ctx, scope.UserID, scope.TaskID)
	if err != nil || task.ProjectID != scope.ProjectID {
		return problem("MoLab access denied for this task.")
	}
	if task.Status == taskdomain.StatusDone || task.Status == taskdomain.StatusFailed || task.ArchivedAt != nil {
		a.molab.Forget(scope.TaskID)
		return problem("This task has ended. Its MoLab pairing was discarded.")
	}
	session, err := a.molab.Get(scope.TaskID, scope.UserID, scope.ProjectID)
	if err != nil {
		return problem("No active MoLab pairing. Ask the user to add a fresh URL and token using the task's secure Connect MoLab form.")
	}
	var input struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(call.Input, &input); err != nil || len(input.Code) > 32000 || strings.TrimSpace(input.Code) == "" {
		return problem("MoLab code must be nonempty Python and at most 32 KB.")
	}
	code := input.Code
	first := !session.Inspected
	if first {
		code = inspectionCode
	}
	// Use the same project's serialized runner. Its one-shot mode does not receive
	// project variables or Git secrets, and cannot be invoked from generic run_command.
	_, err = a.services(scope.ProjectID)
	if err != nil {
		return problem("Project sandbox is unavailable.")
	}
	a.mu.Lock()
	runner := a.runners[scope.ProjectID]
	a.mu.Unlock()
	if runner == nil {
		return problem("Project sandbox is unavailable.")
	}
	command := codingdomain.Task{
		Name:           "marimo-pair",
		Command:        []string{"bash", "/usr/local/share/marimo-pair/scripts/execute-code.sh", "--url", session.URL, "-c", code},
		TimeoutSeconds: 120,
	}
	executionCtx, release := session.Bound(taskContext(ctx, scope))
	defer release()
	result, err := runner.RunMoLab(executionCtx, command, session.Token)
	if err != nil {
		return problem("MoLab execution could not start in the isolated task sandbox.")
	}
	// Never return credentials even if the remote kernel inadvertently echoes them.
	output := strings.ReplaceAll(result.Output, session.Token, "[REDACTED]")
	if result.ExitCode != 0 {
		if strings.Contains(output, "Failed to connect") || strings.Contains(output, "401") || strings.Contains(output, "403") {
			a.molab.Forget(scope.TaskID)
			return problem("MoLab connection was rejected or ended. Pair a fresh notebook using the secure task form.")
		}
		return problem(fmt.Sprintf("MoLab execution failed (exit %d). %s", result.ExitCode, output))
	}
	if first {
		a.molab.MarkInspected(scope.TaskID, session)
		return taskapp.ToolResult{Output: "Connected and inspected marimo._code_mode successfully. Call marimo_execute again with the intended Python. " + output}
	}
	return taskapp.ToolResult{Output: "MoLab Python executed successfully. " + output}
}
