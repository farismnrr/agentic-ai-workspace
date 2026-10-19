package application

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// toolStartSite is the project tool that serves an app to the person. The task module does not import the coding
// module, so the name is repeated here.
const toolStartSite = "start_site"

// maxSecurityRounds bounds how often a start is held back for another review, so a reviewer that keeps finding
// things cannot keep an app from ever starting.
const maxSecurityRounds = 3

// maxArchitectRounds bounds how often a finish is held back for the Architect.
const maxArchitectRounds = 3

var codeExtensions = []string{
	".ts", ".tsx", ".js", ".mjs", ".cjs", ".vue", ".py", ".go", ".rs", ".java", ".kt", ".rb", ".php", ".sql",
}

// securityHold returns why a start_site call must wait, or "" when it may go on. On a team with a Security role,
// an app with code in it does not start until that role has reported on the latest changes.
func (r *Runner) securityHold(ctx context.Context, task domain.Task, call Call) string {
	if r.security == "" || call.Name != toolStartSite {
		return ""
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !changedSinceReview(events, r.security, hasCode, maxSecurityRounds) {
		return ""
	}

	return "The " + r.security + " has not checked the latest code yet, so the app cannot be started. Report back that " +
		"the code is ready for the security check; the orchestrator will ask the " + r.security + " first."
}

// changedSinceReview reports whether files that touches matches were changed after the last time the named reviewer
// reported, and that reviewer has not yet used up its rounds.
func changedSinceReview(events []domain.Event, reviewer string, touches func([]string) bool, maxRounds int) bool {
	var lastChange, lastReview int64

	rounds := 0

	for _, event := range events {
		switch {
		case event.Kind == domain.EventHandoff && event.Agent == reviewer && event.Action == handoffReported:
			lastReview = event.Seq
			rounds++
		case !event.Undone && touches(event.Files):
			lastChange = event.Seq
		}
	}

	return lastChange > lastReview && rounds < maxRounds
}

// inDocuments reports whether a file belongs to the Documents role's output folder. Its scripts and the viewer
// page it generates are not an app, so they do not call for the security check or the design review.
func inDocuments(name string) bool { return strings.HasPrefix(path.Clean(name), "documents/") }

func hasCode(files []string) bool {
	for _, name := range files {
		if inDocuments(name) {
			continue
		}

		base := path.Base(strings.ToLower(name))
		isConfig := base == "package.json" || strings.HasPrefix(base, "nuxt.config.")
		if isConfig || slices.Contains(codeExtensions, path.Ext(base)) {
			return true
		}
	}

	return false
}

// securityOf names the team's Security role, or "" when the team has none.
func securityOf(roster *Roster) string {
	if roster == nil {
		return ""
	}

	for _, member := range roster.Members {
		if member.ChecksSecurity {
			return member.Name
		}
	}

	return ""
}

// architectOf names the team's Architect role, or "" when the team has none.
func architectOf(roster *Roster) string {
	if roster == nil {
		return ""
	}

	for _, member := range roster.Members {
		if member.ChecksArchitecture {
			return member.Name
		}
	}

	return ""
}

// architectNotAsked holds a finish back while code was changed after the Architect last reported: the one who
// knows the architecture verifies the final code before the person is told it is done. At most a few rounds.
func (r *Runner) architectNotAsked(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	architect := architectOf(state.roster)
	if architect == "" || state.architectAsked >= maxArchitectRounds {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !changedSinceReview(events, architect, hasCode, maxArchitectRounds) {
		return false
	}

	state.architectAsked++
	r.result(ctx, task, call.ID, "The "+architect+" has not verified the final code yet. Hand it the changed files "+
		"to check against the design and against SOLID and Clean Architecture, send what it finds back to the "+
		"Developer, and finish when it reports that the code is sound.", true)

	return true
}
