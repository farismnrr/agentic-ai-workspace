package bootstrap

import (
	"context"
	"encoding/json"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	egresscodingtool "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/codingtool"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var toolRequestEgress = codingdomain.ToolRequestEgress.Name

func taskContext(ctx context.Context, scope taskapp.ToolScope) context.Context {
	return codingrunner.WithTask(ctx, scope.ProjectID, scope.TaskID)
}

func (a *agentTools) egressTool() egresscodingtool.Tool {
	if a.egress == nil {
		return egresscodingtool.New(nil)
	}
	return egresscodingtool.New(a.egress.Service())
}

func (a *agentTools) egressDefinitions() []taskdomain.ToolDef {
	return a.egressTool().Definitions()
}

func (a *agentTools) otherTools() map[string]func(context.Context, taskapp.ToolScope, taskapp.Call) taskapp.ToolResult {
	return map[string]func(context.Context, taskapp.ToolScope, taskapp.Call) taskapp.ToolResult{
		toolRequestEgress:     a.executeEgress,
		toolDockerHostStatus:  a.executeDockerHostStatus,
		toolMoLabExecute:      a.executeMoLab,
		toolScreenshot:        a.executeShot,
		toolScreenshotSession: a.executeScreenshotSession,
		toolDeployExtension: func(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
			return a.executeExtension(ctx, scope.ProjectID, call)
		},
		toolCILogs: a.executeCILogs,
		toolViewImage: func(_ context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
			return a.executeView(scope.ProjectID, call)
		},
	}
}

func describeEgress(input json.RawMessage) taskapp.Described {
	return egresscodingtool.New(nil).Describe(input)
}

func (a *agentTools) executeEgress(
	ctx context.Context, scope taskapp.ToolScope, call taskapp.Call,
) taskapp.ToolResult {
	return a.egressTool().Execute(ctx, scope, call)
}
