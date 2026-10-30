package bootstrap

import (
	"context"

	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// chosenWindow is the context window the person set for the model of this turn; zero leaves it to the program.
func (p *plans) chosenWindow(ctx context.Context, req taskapp.CLIRequest) int64 {
	if p.windows == nil {
		return 0
	}

	model := req.Model
	if model == "" {
		model = cliagentdomain.DefaultModel
	}

	sizes, err := p.windows.All(ctx, req.UserID, req.Program)
	if err != nil {
		return 0
	}

	return sizes[model]
}

// catalogWindow is a real model-reported size for the selected vendor model;
// unknown/default aliases are left unknown rather than inventing 200K.
func catalogWindow(models []cliagentdomain.Model, selected string) int64 {
	if selected == "" || selected == cliagentdomain.DefaultModel {
		return 0
	}
	for _, model := range models {
		if model.ID == selected {
			return model.ContextWindow
		}
	}
	return 0
}

// shownWindow uses the selected limit, capped to a reported model limit when available.
func shownWindow(chosen, reported int64) int64 {
	if chosen <= 0 {
		return reported
	}

	if reported > 0 {
		return min(chosen, reported)
	}

	return chosen
}
