package application

import (
	"context"
	"regexp"
	"slices"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxCommitHolds is how often a finish is held back for the work to be saved in git.
const maxCommitHolds = 3

// saved matches the command that saves the work: the sandbox's ship, or a plain git commit.
var saved = regexp.MustCompile(`(^\s*ship\b|\bgit\s+commit\b)`)

// committerOf names the teammate that saves the work in git: the one that starts the web app, DevOps on the
// built-in team. A team without one saves nothing and is never held.
func committerOf(roster *Roster) string {
	if roster == nil {
		return ""
	}

	for _, member := range roster.Members {
		if slices.Contains(member.Tools, toolStartSite) {
			return member.Name
		}
	}

	return ""
}

// commitMissing holds a finish back while files changed after the work was last saved in git. The commit and the
// push come last, after the checks, so the person is told only about work that is already saved.
func (r *Runner) commitMissing(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	committer := committerOf(state.roster)
	if committer == "" || state.commitAsked >= maxCommitHolds {
		return false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !changedSinceCommit(events) {
		return false
	}

	state.commitAsked++
	r.result(ctx, task, call.ID, "The work is not saved in git yet. Before you report to the person, give the "+committer+
		" the job to commit and push it with ship (it knows how, and tells you whether the push worked). The "+
		"push may fail without the job failing; say so plainly in your summary.", true)

	return true
}

// changedSinceCommit reports whether a file changed after the last commit that succeeded.
func changedSinceCommit(events []domain.Event) bool {
	var change, commit int64

	for _, event := range events {
		switch {
		case isCommit(event):
			commit = event.Seq
		case !event.Undone && len(event.Files) > 0:
			change = event.Seq
		}
	}

	return change > commit
}

// isCommit reports whether an event is the command that saved the work, and it worked.
func isCommit(event domain.Event) bool {
	return event.Kind == domain.EventAction && event.Run != nil && event.Run.ExitCode == 0 &&
		saved.MatchString(event.Run.Command)
}
