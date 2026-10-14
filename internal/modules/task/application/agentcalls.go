package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (r *Runner) askUser(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	args, ok := parseAsk(call.Input)
	if !ok {
		r.result(ctx, task, call.ID, "ask_user needs a question.", true)

		return outcome{}, false
	}

	event, found, err := r.deps.Events.EventByToolUse(ctx, task.ID, call.ID)
	if err != nil {
		return outcome{stop: stopWaiting}, true
	}

	if !found {
		_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
			Kind: domain.EventQuestion, Text: args.Question, Options: args.Options, ToolUseID: call.ID,
			Time: r.deps.Now(),
		})
		_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusNeedsInput, r.deps.Now())

		return outcome{stop: stopWaiting}, true
	}

	if event.Answered == "" {
		return outcome{stop: stopWaiting}, true
	}

	r.result(ctx, task, call.ID, "The person answered: "+event.Answered, false)

	return outcome{}, false
}

func (r *Runner) updatePlan(ctx context.Context, task domain.Task, call Call) {
	steps, ok := parsePlan(call.Input)
	if !ok {
		r.result(ctx, task, call.ID, "update_plan needs 1 to 12 steps, each with a title and a state.", true)

		return
	}

	_ = r.deps.Tasks.SavePlan(ctx, task.ID, steps, r.deps.Now())
	_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, r.deps.Now())
	r.result(ctx, task, call.ID, "Plan saved.", false)
}

func (r *Runner) finish(ctx context.Context, task domain.Task, call Call) outcome {
	args := parseFinish(call.Input)
	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	r.result(ctx, task, call.ID, "Recorded. The person will review your work.", false)

	return r.complete(ctx, task, args.Summary, buildProof(events, args.Checks))
}

// complete hands the task to the person for review with its summary and proof.
func (r *Runner) complete(ctx context.Context, task domain.Task, summary string, proof *domain.Proof) outcome {
	if summary == "" {
		summary = "Done."
	}

	receipt := domain.Event{Kind: domain.EventReceipt, Text: summary, Time: r.deps.Now()}
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, receipt)
	_ = r.deps.Tasks.Finish(ctx, task.ID, summary, proof, r.deps.Now())

	return outcome{stop: stopFinished}
}

// skip answers calls that came after finish so the conversation stays valid.
func (r *Runner) skip(ctx context.Context, task domain.Task, calls []Call) {
	for _, call := range calls {
		r.result(ctx, task, call.ID, "Skipped: the task was already finished.", true)
	}
}
