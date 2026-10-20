package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxSiteRefusals is how many times a finish is held back for a missing running app. It is a limit so that an
// app that cannot start does not stop the run from ever ending.
const maxSiteRefusals = 3

// pageBuiltButNotRunning reports whether the work made a web page and no app has been started since.
func pageBuiltButNotRunning(events []domain.Event) bool {
	built := false

	for _, event := range events {
		switch {
		case event.Kind == domain.EventSite:
			if built {
				return false
			}
		case !event.Undone && hasPage(event.Files):
			built = true
		}
	}

	return built
}

func hasPage(files []string) bool {
	for _, name := range files {
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") || strings.HasSuffix(lower, ".vue") {
			return true
		}
	}

	return false
}

// siteNotRunning holds a finish back while the work includes a page nobody can open yet. The person reads the
// result in the browser, so the app must be running before the report reaches them. It gives up after a few
// refusals, so a page that cannot be served never blocks the end of a run.
func (r *Runner) siteNotRunning(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	return r.holdFinish(ctx, task, call, &state.siteAsked, maxSiteRefusals, pageBuiltButNotRunning,
		"The work includes a web page, but no app is running for the person to open. Start it first with start_site "+
			"(serve the project folder, for example python3 -m http.server on a port), check that it answers, and then "+
			"finish. On a team, only the DevOps teammate can start it: give it that job.")
}

// tryFinish ends the run unless the plan is out of date or a page the person cannot open is waiting to be served.
func (r *Runner) tryFinish(ctx context.Context, task domain.Task, state *runState, call Call) (outcome, bool) {
	if r.planStale(ctx, task, state, call) || r.siteNotRunning(ctx, task, state, call) ||
		r.localAddressHeld(ctx, task, state, call) || r.extensionHeld(ctx, task, state, call) ||
		r.securityNotAsked(ctx, task, state, call) || r.architectNotAsked(ctx, task, state, call) ||
		r.reviewerNotAsked(ctx, task, state, call) || r.commitMissing(ctx, task, state, call) {
		return outcome{}, false
	}

	return r.finish(ctx, task, call), true
}
