package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	// nudgeAfter is how many steps in a row may fail before a teammate is told to stop and report.
	nudgeAfter = 3
	// giveUpAfter is how many may fail in a row before the server ends the job itself and sends it back.
	giveUpAfter   = 6
	maxStuckError = 600

	stuckNudge = "Several steps in a row did not work. Stop trying variations of the same thing: call report now " +
		"with a summary starting \"Stuck:\" for a real repeated failure, or \"Needs <role>:\" for a normal " +
		"missing prerequisite owned by a teammate; include what you tried and the exact error, " +
		"and the question that someone who can search the web should answer for you."
	// stuckAdvice goes after a teammate's failed job, so the orchestrator hands the problem to the Researcher.
	stuckAdvice = " Do not repeat the same job unchanged. If a prerequisite belongs to another role, " +
		"delegate to that owner and then resume. For a genuine technical unknown, ask Researcher to investigate " +
		"the exact failure with sources before retrying."
)

// trackMisses counts the steps of a teammate that failed in a row, so one that is going in circles is told to
// ask for help and, failing that, is stopped. A teammate that gets past the problem starts counting again.
func (r *Runner) trackMisses(
	ctx context.Context, task domain.Task, state *runState, result ToolResult,
) (outcome, bool) {
	if r.role == nil || state == nil {
		return outcome{}, false
	}

	if !result.IsError {
		state.misses = 0

		return outcome{}, false
	}

	state.misses++

	switch state.misses {
	case nudgeAfter:
		_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser,
			domain.Block{Kind: domain.BlockText, Text: stuckNudge})
	case giveUpAfter:
		return failed(fmt.Sprintf("Stuck: %d steps in a row did not work. The last problem: %s",
			giveUpAfter, truncate(result.Output, maxStuckError)), ""), true
	}

	return outcome{}, false
}
