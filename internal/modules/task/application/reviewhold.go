package application

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxLookRounds bounds how often a finish is held back for the Reviewer, and how often a Designer is sent back to
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

// reviewerOf names the team's Reviewer, the role that must look at the screens before the task is done.
func reviewerOf(roster *Roster) string {
	if roster == nil {
		return ""
	}

	for _, member := range roster.Members {
		if member.ChecksLook {
			return member.Name
		}
	}

	return ""
}

// reviewerNotAsked holds a finish back while screens changed after the Reviewer last reported. The look is judged by
// someone who is not the one who made it, on the running app, before the person is told it is done.
func (r *Runner) reviewerNotAsked(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	reviewer := reviewerOf(state.roster)
	if reviewer == "" || state.reviewerAsked >= maxLookRounds {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !changedSinceReview(events, reviewer, hasScreen, maxLookRounds) {
		return false
	}

	state.reviewerAsked++
	r.result(ctx, task, call.ID, "The "+reviewer+" has not looked at the screens yet. Make sure the app is running, then "+
		"hand the "+reviewer+" a brief that says \"design review\" and names DESIGN.md and the pages. Send what it "+
		"finds to the Designer or the Developer, and finish when it reports that the product looks right.", true)

	return true
}

// madeAndLooked reports whether a teammate changed screens, and whether it took a screenshot, in the events given.
func madeAndLooked(events []domain.Event, name string) (changed, looked bool) {
	for _, event := range events {
		if event.Agent != name {
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
	if r.role == nil || !r.role.LooksAtScreens || state.lookAsked >= maxLookRounds {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, r.since)
	if changed, looked := madeAndLooked(events, r.role.Name); !changed || looked {
		return false
	}

	state.lookAsked++
	r.result(ctx, task, call.ID, "You changed screens but have not looked at them. Start the app (start_site), then "+
		"look at every page you made with screenshot: desktop first, then device mobile. Fix what you see, look "+
		"again, and then report.", true)

	return true
}
