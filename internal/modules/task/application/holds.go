package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// holds counts how often the server held something back during one run, so that every hold is bounded and a
// check that cannot be satisfied never stops the run from ending.
type holds struct {
	// planWarned is set once the orchestrator was told its plan was out of date, so it is told only once.
	planWarned bool
	// siteAsked counts finishes held back because a page was built and no app was running.
	siteAsked int
	// localAsked counts finishes sent back for naming 127.0.0.1 or localhost.
	localAsked int
	// extensionAsked counts finishes held back until a browser extension is packaged.
	extensionAsked int
	// commitAsked counts finishes held back until the work is saved in git.
	commitAsked int
	// clarifyAsked counts jobs held back so the orchestrator decides whether to ask the person first.
	clarifyAsked int
	// planAsked counts jobs held back because the person's latest request was not in the plan yet.
	planAsked int
	// securityAsked counts finishes held back for the Security check of the final code.
	securityAsked int
	// architectAsked counts finishes held back for the Architect's check of the final code.
	architectAsked int
	// reviewerAsked counts finishes held back for the Reviewer's look at the screens.
	reviewerAsked int
	// lookAsked counts reports sent back because a teammate had not looked at the screens it made.
	lookAsked int
}

// holdFinish holds a finish back with text while unmet says the thread still lacks something. It counts the holds in
// asked and gives up after max, so a check that cannot be met never keeps a run from ending.
func (r *Runner) holdFinish(
	ctx context.Context, task domain.Task, call Call, asked *int, max int, unmet func([]domain.Event) bool, text string,
) bool {
	if *asked >= max {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !unmet(events) {
		return false
	}

	*asked++

	r.result(ctx, task, call.ID, text, true)

	return true
}
