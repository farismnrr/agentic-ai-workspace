package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (r *Runner) askUser(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	questions, ok := parseAsk(call.Input)
	if !ok {
		r.result(ctx, task, call.ID, "ask_user needs a question.", true)

		return outcome{}, false
	}

	event, found, err := r.deps.Events.EventByToolUse(ctx, task.ID, call.ID)
	if err != nil {
		return outcome{stop: stopWaiting}, true
	}

	if !found {
		if r.askedEnough(ctx, task, len(questions)) {
			r.result(ctx, task, call.ID, "You have asked enough questions. Take a sensible default for the rest, "+
				"list it under assumptions, and finish.", true)

			return outcome{}, false
		}

		_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
			Kind: domain.EventQuestion, Text: questions[0].Text, Options: optionLabels(questions[0]),
			Questions: questions, ToolUseID: call.ID, Time: r.deps.Now(),
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

// maxTeammateQuestions is how many questions one teammate may put to the person on a task: they are there to
// settle what blocks the work, not to interview.
const maxTeammateQuestions = 6

// askedEnough reports whether a form of this many questions would take the teammate past its limit.
func (r *Runner) askedEnough(ctx context.Context, task domain.Task, adding int) bool {
	if r.role == nil {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	asked := 0

	for _, event := range events {
		if event.Kind == domain.EventQuestion && event.Agent == r.role.Name {
			asked += max(len(event.Questions), 1)
		}
	}

	return asked+adding > maxTeammateQuestions
}

func (r *Runner) updatePlan(ctx context.Context, task domain.Task, call Call) {
	steps, ok := parsePlan(call.Input)
	if !ok {
		r.result(ctx, task, call.ID, "update_plan needs 1 to 40 steps, each with a title and a state.", true)

		return
	}

	if err := r.deps.Tasks.SavePlan(ctx, task.ID, steps, r.deps.Now()); err != nil {
		r.result(ctx, task, call.ID, "The plan could not be saved. Try update_plan again.", true)
		return
	}
	_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, r.deps.Now())
	r.recordPlan(ctx, task, steps)
	if err := r.saveNotes(ctx, task); err != nil {
		r.result(ctx, task, call.ID, "The plan is in the task, but its project file could not be saved. Try again.", true)
		return
	}
	r.result(ctx, task, call.ID, "Plan saved.", false)
}

func (r *Runner) finish(ctx context.Context, task domain.Task, call Call) outcome {
	args := parseFinish(call.Input)
	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)

	if !workDone(events) {
		r.result(ctx, task, call.ID, "Recorded.", false)

		return r.answer(ctx, task, args.Summary)
	}

	r.result(ctx, task, call.ID, "Recorded. The person will review your work.", false)

	return r.complete(ctx, task, args.Summary, buildProof(events, args.Checks))
}

// answer ends a run that changed nothing: the reply is plain conversation, so there is
// no receipt and nothing to review. The thread stays open for the next message.
func (r *Runner) answer(ctx context.Context, task domain.Task, text string) outcome {
	if text == "" {
		text = "Done."
	}

	r.say(ctx, task.ID, text)
	_ = r.deps.Tasks.Finish(ctx, task.ID, text, nil, r.deps.Now())
	_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusDone, r.deps.Now())

	return outcome{stop: stopFinished}
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
