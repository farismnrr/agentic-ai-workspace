package bootstrap

import (
	"context"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// The host probe exposes one fixed read-only operation through the trusted app.
// It is not a Docker CLI tunnel and does not grant access to the Docker socket.
const toolDockerHostStatus = "docker_host_status"

func dockerHostStatusDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name:        toolDockerHostStatus,
		Description: "Check whether AI Code's app can reach its host Docker daemon. Read-only, no arguments, and does not expose the socket to the task sandbox. If unavailable, ask the operator to run scripts/check-docker-host.sh on the actual host.",
		Schema:      []byte(`{"type":"object","properties":{},"additionalProperties":false}`),
	}
}

func (a *agentTools) executeDockerHostStatus(ctx context.Context, _ taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if len(call.Input) != 0 && string(call.Input) != "{}" {
		return problem("Docker host status takes no arguments.")
	}
	if err := sandbox.ProbeHost(ctx); err != nil {
		return problem("Docker host: unavailable to the app. Ask the operator to run 'bash scripts/check-docker-host.sh dev' on the Docker-enabled host. No sandbox permission change is needed.")
	}

	return taskapp.ToolResult{Output: "Docker host: reachable from the AI Code app. This is a read-only daemon ping; the sandbox still has no Docker socket. Image availability and container execution are not verified by this check."}
}
