package bootstrap

import (
	"context"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	molabapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
)

// projectMoLab reuses the project's existing Docker runner, but it cannot load
// project secrets or grant host access beyond the validated notebook hostname.
type projectMoLab struct {
	runner *codingrunner.Runner
}

func (p projectMoLab) Execute(ctx context.Context, url, token, code string) codingmcp.MoLabResult {
	result := molabapp.Execute(ctx, p.runner, url, token, code)
	return codingmcp.MoLabResult{Output: result.Output, Problem: result.Problem}
}
