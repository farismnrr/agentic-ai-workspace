package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxFactName = 80

// facts is what the server itself recorded for a teammate: the files they changed and the checks they
// ran. It is appended to their report, so the orchestrator weighs what they say against what happened.
func (r *Runner) facts(ctx context.Context, task domain.Task, d Delegation) string {
	events, err := r.deps.Events.EventsAfter(ctx, task.ID, d.StartSeq)
	if err != nil {
		return ""
	}

	files, seen, runs := []string{}, map[string]bool{}, []string{}

	for _, event := range events {
		if !belongsToJob(event, d.RoleName, d.ID) || event.Undone {
			continue
		}

		for _, file := range event.Files {
			if !seen[file] {
				seen[file] = true
				files = append(files, file)
			}
		}

		if event.Run != nil {
			runs = append(runs, fmt.Sprintf("%s (%s)", truncate(event.Run.Name, maxFactName), runResult(*event.Run)))
		}
	}

	return "\n\nWhat the server recorded for " + d.RoleName + ": " + describeWork(files, runs) + "."
}

func belongsToJob(event domain.Event, roleName, jobID string) bool {
	if jobID != "" && event.Job != "" {
		return event.Job == jobID
	}

	return event.Agent == roleName
}

func runResult(run domain.RunInfo) string {
	switch {
	case run.TimedOut:
		return "timed out"
	case passedRun(run):
		return "passed"
	}

	return fmt.Sprintf("failed, exit %d", run.ExitCode)
}

func describeWork(files, runs []string) string {
	parts := []string{}

	if len(files) > 0 {
		parts = append(parts, "changed "+strings.Join(files, ", "))
	}

	if len(runs) > 0 {
		parts = append(parts, "ran "+strings.Join(runs, ", "))
	}

	if len(parts) == 0 {
		return "no files were changed and no checks were run"
	}

	return strings.Join(parts, "; ")
}
