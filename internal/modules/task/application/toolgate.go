package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const toolOffText = "This tool is turned off in Settings, Spaces & tools. " +
	"Try another way, or ask the person to turn it on."

// offered lists the project tools the person switched on, plus the agent's own loop tools: the
// orchestrator also gets delegate, a teammate only report.
func (r *Runner) offered(ctx context.Context, userID string, state *runState) ([]domain.ToolDef, error) {
	enabled, err := r.deps.Gate.Enabled(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check tool switches: %w", err)
	}

	defs := []domain.ToolDef{}

	for _, def := range r.deps.Tools.Definitions() {
		if enabled[def.Name] {
			defs = append(defs, def)
		}
	}

	switch {
	case r.role != nil:
		return append(defs, reportTool()), nil
	case state.roster != nil:
		return append(append(defs, agentTools()...), delegateTool(state.roster.Members)), nil
	}

	return append(defs, agentTools()...), nil
}

// toolOn re-checks a switch right before a call runs, since it may have changed mid-task.
// When the switch cannot be read the tool stays off.
func (r *Runner) toolOn(ctx context.Context, userID, tool string) bool {
	enabled, err := r.deps.Gate.Enabled(ctx, userID)

	return err == nil && enabled[tool]
}

// skipsApprovals reports whether the person lets the agent act without asking. When the setting
// cannot be read the agent asks.
func (r *Runner) skipsApprovals(ctx context.Context, userID string) bool {
	skip, err := r.deps.Gate.SkipApprovals(ctx, userID)

	return err == nil && skip
}
