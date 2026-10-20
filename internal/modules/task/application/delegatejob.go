package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// doJob has the teammate work on the job: with the app's own loop, or with the person's own plan when that is
// the model they were given.
func (r *Runner) doJob(
	ctx context.Context, worker *Runner, teammate domain.Task, sub *runState, role AgentRole, d Delegation,
) outcome {
	if onPlan(d.ProviderID) {
		return r.workOnPlan(ctx, worker, teammate, role, d)
	}

	return worker.drive(ctx, teammate, sub, stepsText("The "+d.RoleName, r.lim.delegateSteps()))
}
