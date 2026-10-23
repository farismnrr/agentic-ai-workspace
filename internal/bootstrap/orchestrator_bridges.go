package bootstrap

import (
	"context"
	"fmt"

	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	orchestrator "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/mcp"
)

type orchestratorGate struct{ permissions *spaceapp.Permissions }

func (g orchestratorGate) Allowed(ctx context.Context, userID, tool string) (bool, error) {
	enabled, err := g.permissions.Enabled(ctx, userID, "orchestrator", tool)
	if err != nil {
		return false, fmt.Errorf("check Orchestrator tool switch: %w", err)
	}
	return enabled, nil
}

type orchestratorActivity struct{ log *spaceapp.ActivityLog }

func (a orchestratorActivity) Record(ctx context.Context, entry orchestrator.Entry) error {
	_, err := a.log.Record(ctx, spacedomain.Activity{
		UserID: entry.UserID, Space: "orchestrator", Client: entry.Client, Tool: entry.Tool,
		Summary: entry.Summary, Detail: entry.Detail, Outcome: spacedomain.Outcome(entry.Outcome),
		ProjectID: entry.Project, TaskID: entry.Task,
	})
	if err != nil {
		return fmt.Errorf("record Orchestrator activity: %w", err)
	}
	return nil
}
