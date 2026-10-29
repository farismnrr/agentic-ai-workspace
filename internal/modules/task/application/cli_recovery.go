package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const freshCLINote = "The previous AI context reached its limit. Continue in a fresh vendor session using the task history. Inspect the workspace before changing anything, and do not repeat completed changes."

// isCLIContextOverflow only handles vendor context-length errors, not general failures.
func isCLIContextOverflow(reason string) bool {
	text := strings.ToLower(reason)
	for _, phrase := range overflowWords {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

// needsFreshCLISession uses the last measured occupancy before invoking a
// resumed vendor session. Zero usage means unknown, never "empty".
func needsFreshCLISession(task domain.Task, session string) bool {
	return session != "" && (task.ContextState == "failed" ||
		task.ContextWindow > 0 && task.ContextUsed > 0 && task.ContextUsed >= compactThreshold(task.ContextWindow))
}

// freshCLIRequest forks a session without deleting the old vendor conversation.
// The old session ID is only replaced after the new request succeeds.
func (r *Runner) freshCLIRequest(ctx context.Context, task domain.Task, req CLIRequest) CLIRequest {
	req.Session = ""
	history := r.continuation(ctx, task.ID)
	const historyBudget = 12_000
	if len(history) > historyBudget {
		history = "Earlier work (most recent excerpt):\n" + strings.ToValidUTF8(history[len(history)-historyBudget:], "")
	}
	req.Prompt = freshCLINote + "\n" + history + "\n\nCurrent request:\n" + req.Prompt
	return req
}

// runCLIWithRecovery preflights and attempts at most one safe rollover.
func (r *Runner) runCLIWithRecovery(ctx context.Context, task domain.Task, req CLIRequest, sink *planSink) (CLIResult, *planSink, bool, error) {
	fresh := needsFreshCLISession(task, req.Session)
	if fresh {
		r.say(ctx, task.ID, "Context reached its limit; starting a fresh vendor session with recent task history.")
		req = r.freshCLIRequest(ctx, task, req)
	}
	if sink.runner.role == nil {
		_ = sink.runner.deps.Tasks.SetContext(ctx, task.ID, 0, 0)
		if fresh {
			sink.runner.setCompactionState(ctx, task, "compacting")
		}
	}
	result, err := r.deps.CLI.Run(ctx, req, func(event CLIEvent) { sink.emit(ctx, event) })
	if ctx.Err() != nil {
		if fresh {
			sink.runner.setCompactionState(context.WithoutCancel(ctx), task, "failed")
		}
		return result, sink, fresh, err
	}
	failure := result.Reason
	if err != nil {
		failure = err.Error()
	}
	if req.Session != "" && !sink.didStep && isCLIContextOverflow(failure) {
		r.say(ctx, task.ID, "Vendor rejected the full context; retrying once in a fresh session.")
		req = r.freshCLIRequest(ctx, task, req)
		fresh = true
		sink.runner.setCompactionState(ctx, task, "compacting")
		sink = &planSink{runner: sink.runner, task: task}
		result, err = r.deps.CLI.Run(ctx, req, func(event CLIEvent) { sink.emit(ctx, event) })
	}
	finalFailure := result.Reason
	if err != nil {
		finalFailure = err.Error()
	}
	if (fresh && finalFailure != "") || isCLIContextOverflow(finalFailure) {
		sink.runner.setCompactionState(ctx, task, "failed")
	} else if finalFailure == "" {
		// A successful rollover must leave the transient "compacting" state.
		// Prefer the last-call meter; if unavailable, retain a valid streamed
		// measurement, otherwise clear the stale pre-compaction usage.
		if result.Used > 0 && result.Window > 0 {
			sink.runner.saveMeter(ctx, task, result)
		} else if fresh && !sink.meterKnown && sink.runner.role == nil {
			_ = sink.runner.deps.Tasks.SetContext(ctx, task.ID, 0, 0)
		}
	}
	return result, sink, fresh, err
}
