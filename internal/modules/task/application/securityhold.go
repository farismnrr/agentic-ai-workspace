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
	if !codeChangedSinceReview(events, r.security) {
		return ""
	}

	return "The " + r.security + " has not checked the latest code yet, so the app cannot be started. Report back that " +
		"the code is ready for the security check; the orchestrator will ask the " + r.security + " first."
}

// codeChangedSinceReview reports whether code was changed after the last time the Security role reported, and that
// role has not yet used up its rounds.
func codeChangedSinceReview(events []domain.Event, security string) bool {
	var lastChange, lastReview int64

	rounds := 0

	for _, event := range events {
		switch {
		case event.Kind == domain.EventHandoff && event.Agent == security && event.Action == handoffReported:
			lastReview = event.Seq
			rounds++
		case !event.Undone && hasCode(event.Files):
			lastChange = event.Seq
		}
	}

	return lastChange > lastReview && rounds < maxSecurityRounds
}

func hasCode(files []string) bool {
	for _, name := range files {
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
