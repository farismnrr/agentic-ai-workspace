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

// shownWindow is the window the meter counts against: the size the person chose, held to what the program says
// the model can really take.
func shownWindow(chosen, reported int64) int64 {
	if chosen <= 0 {
		return reported
	}

	if reported > 0 {
		return min(chosen, reported)
	}

	return chosen
}
