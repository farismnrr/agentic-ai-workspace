package application

import (
	"context"
	"encoding/json"
	"path"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxLookRounds bounds how often a finish is held back for QA review, and how often Developer is sent back to
// look at its work.
const maxLookRounds = 3

var screenExtensions = []string{
	".html", ".htm", ".vue", ".svelte", ".tsx", ".jsx", ".css", ".scss", ".ejs", ".hbs", ".njk", ".pug",
}

// hasScreen reports whether the files include something the person sees: a page, a component or a style.
func hasScreen(files []string) bool {
	for _, name := range files {
		if !inDocuments(name) && slices.Contains(screenExtensions, path.Ext(strings.ToLower(name))) {
			return true
		}
	}

	return false
}

// reviewerOf resolves the QA teammate responsible for independent review.
func reviewerOf(roster *Roster) AgentRole {
	if roster == nil {
		return AgentRole{}
	}

	for _, member := range roster.Members {
		if member.ChecksLook {
			return member
		}
	}

	return AgentRole{}
}

// reviewerNotAsked requires independent review only when the saved impact policy calls for it.
func (r *Runner) reviewerNotAsked(
	ctx context.Context, task domain.Task, state *runState, call Call, impact verificationImpact,
) bool {
	if !impact.reviewer {
		return false
	}
	return r.requireReview(ctx, task, state, call, reviewerOf(state.roster), &state.reviewerAsked,
		maxLookRounds, hasImpactChange, "QA", impact.reason,
		"Independently review the changed behavior and relevant regression evidence. Do not assume a passing report.")
}

// madeAndLooked reports whether a teammate changed screens, and whether it took a screenshot, in the events given.
func madeAndLooked(events []domain.Event, name, jobID string) (changed, looked bool) {
	for _, event := range events {
		if !belongsToJob(event, name, jobID) {
			continue
		}

		changed = changed || (!event.Undone && hasScreen(event.Files))
		looked = looked || event.Kind == domain.EventShot
	}

	return changed, looked
}

// lookNotTaken sends a teammate who makes screens back to look at the running app before it reports: its own steps
// since it began changed screens, and none of them was a screenshot. Told at most a few times, so an app that cannot
// be shown does not stop the report for ever.
func (r *Runner) lookNotTaken(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	if r.role == nil || !r.role.LooksAtScreens || state.lookAsked >= maxLookRounds || asksForApp(call.Input) {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, r.since)
	if changed, looked := madeAndLooked(events, r.role.Name, r.job); !changed || looked {
		return false
	}

	state.lookAsked++
	r.result(ctx, task, call.ID, "You changed screens but have not looked at them. You cannot start the app: the "+
		"DevOps teammate does. If it is not running, report \"Needs the app started\". Once it runs, look at every "+
		"page you made with screenshot: desktop first, then device mobile. Fix what you see, look again, and then "+
		"report.", true)

	return true
}

// asksForApp reports whether a report says the app must be started first: only DevOps can, so that is not a
// teammate who skipped looking.
func asksForApp(input json.RawMessage) bool {
	text, _, ok := parseReport(input)

	return ok && strings.Contains(strings.ToLower(text), "needs the app started")
}
