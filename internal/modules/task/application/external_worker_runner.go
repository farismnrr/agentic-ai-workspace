package application

import "context"

type cancellationAwareTools struct {
	ToolSet
}

func (t cancellationAwareTools) Execute(ctx context.Context, scope ToolScope, call Call) ToolResult {
	if err := ctx.Err(); err != nil {
		return ToolResult{Output: err.Error(), IsError: true}
	}
	return t.ToolSet.Execute(ctx, scope, call)
}

func externalWorkerRunner(r *Runner) *Runner {
	worker := *r
	worker.externalFacts = true
	worker.externalWorker = true
	worker.deps.Tools = cancellationAwareTools{ToolSet: r.deps.Tools}
	return &worker
}
