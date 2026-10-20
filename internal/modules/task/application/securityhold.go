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

// maxSecurityRounds bounds how often a finish is held back for another security check, so a reviewer that keeps
// finding things cannot keep the run from ever ending.
const maxSecurityRounds = 3

// maxArchitectRounds bounds how often a finish is held back for the Architect.
const maxArchitectRounds = 3

var codeExtensions = []string{
	".ts", ".tsx", ".js", ".mjs", ".cjs", ".vue", ".py", ".go", ".rs", ".java", ".kt", ".rb", ".php", ".sql",
}

// securityNotAsked holds a finish back while code was changed after the Security role last reported. The app can be
// started and restarted freely while it is being built (a private site only the owner opens); the check of the
// finished code happens once, before the person is told it is done. At most a few rounds.
func (r *Runner) securityNotAsked(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	security := securityOf(state.roster)

	return r.finalCheckMissing(ctx, task, call, security, &state.securityAsked, maxSecurityRounds,
		"The "+security+" has not checked the final code yet. Hand it the changed files once, now that the work is "+
			"done, send its \"Fix first\" findings back to the Developer, and finish when it says the code is safe.")
}

// finalCheckMissing holds a finish back, with text as the answer, while code changed after reviewer last reported.
// It counts the holds in asked and gives up after maxRounds, so a check that cannot be met never blocks the end.
func (r *Runner) finalCheckMissing(
	ctx context.Context, task domain.Task, call Call, reviewer string, asked *int, maxRounds int, text string,
) bool {
	if reviewer == "" || *asked >= maxRounds {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !changedSinceReview(events, reviewer, hasCode, maxRounds) {
		return false
	}

	*asked++

	r.result(ctx, task, call.ID, text, true)

	return true
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

	return r.finalCheckMissing(ctx, task, call, architect, &state.architectAsked, maxArchitectRounds,
		"The "+architect+" has not verified the final code yet. Hand it the changed files to check against the design "+
			"and against SOLID and Clean Architecture, send what it finds back to the Developer, and finish when it "+
			"reports that the code is sound.")
}
