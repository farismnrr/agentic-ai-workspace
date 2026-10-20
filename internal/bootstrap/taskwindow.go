package bootstrap

import (
	"context"
	"strconv"

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

// windowEnv tells Claude Code the window to shorten its notes by; ChatGPT's program takes it as an option.
func windowEnv(program string, window int64) []string {
	if program != string(cliagentdomain.Claude) || window <= 0 {
		return nil
	}

	return []string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW=" + strconv.FormatInt(window, 10)}
}

func codexArgv(req taskapp.CLIRequest, model, effort string, window int64) []string {
	argv := []string{"codex", "exec"}
	if req.Session != "" {
		argv = append(argv, "resume")
	}

	argv = append(argv, "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox")
	argv = append(argv, choice("-m", model, "-c", `model_reasoning_effort="`+effort+`"`, effort)...)

	if window > 0 {
		argv = append(argv, "-c", "model_context_window="+strconv.FormatInt(window, 10))
	}

	if req.Bridge != nil {
		argv = append(argv, bridgeArgs(cliagentdomain.ChatGPT)...)
	}

	if req.Session != "" {
		argv = append(argv, req.Session)
	}

	return append(argv, "-")
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
