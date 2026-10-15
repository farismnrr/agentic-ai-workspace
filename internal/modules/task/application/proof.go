package application

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// buildProof derives the evidence for a finished task. Checks come from tasks
// the server ran itself after the last change ("verified"); anything the agent
// only says it checked is listed as claimed and never counts as proof.
func buildProof(events []domain.Event, claims []string) *domain.Proof {
	runs := runsSinceLastChange(events)
	proof := &domain.Proof{Checks: []domain.Check{}}
	failed := false

	for _, run := range runs {
		passed := passedRun(run)
		failed = failed || !passed
		proof.Checks = append(proof.Checks, domain.Check{Text: runText(run, passed), Verified: true, Passed: passed})
	}

	for _, claim := range claims {
		proof.Checks = append(proof.Checks, domain.Check{Text: claim})
	}

	proof.Proven = len(runs) > 0 && !failed
	proof.Rows = []domain.ProofRow{
		{Label: "What changed", Value: changedSummary(events)},
		{Label: "How it was tested", Value: testedSummary(runs)},
		{Label: "Result", Value: resultText(runs, failed)},
	}

	return proof
}

func runsSinceLastChange(events []domain.Event) []domain.RunInfo {
	start := 0

	for i, event := range events {
		if len(event.Files) > 0 && !event.Undone {
			start = i + 1
		}
	}

	runs := []domain.RunInfo{}

	for _, event := range events[start:] {
		if event.Run != nil {
			runs = append(runs, *event.Run)
		}
	}

	return runs
}

func runText(run domain.RunInfo, passed bool) string {
	switch {
	case run.TimedOut:
		return fmt.Sprintf("The %q task ran out of time", run.Name)
	case passed:
		return fmt.Sprintf("The %q task passed", run.Name)
	}

	return fmt.Sprintf("The %q task failed", run.Name)
}

func changedSummary(events []domain.Event) string {
	files := map[string]bool{}

	for _, event := range events {
		if event.Undone {
			continue
		}

		for _, file := range event.Files {
			files[file] = true
		}
	}

	switch len(files) {
	case 0:
		return "No files were changed"
	case 1:
		return "1 file changed"
	}

	return fmt.Sprintf("%d files changed", len(files))
}

// passedRun reports whether a check the server ran succeeded.
func passedRun(run domain.RunInfo) bool {
	return run.ExitCode == 0 && !run.TimedOut
}

// testedSummary says how many checks ran and how they ended. It never lists the commands: they are long, and the
// "What was checked" list under it already names each one.
func testedSummary(runs []domain.RunInfo) string {
	if len(runs) == 0 {
		return "Nothing was run to check it"
	}

	passed := 0

	for _, run := range runs {
		if passedRun(run) {
			passed++
		}
	}

	failed := len(runs) - passed

	switch {
	case len(runs) == 1 && failed == 0:
		return "Ran 1 check, and it passed"
	case len(runs) == 1:
		return "Ran 1 check, and it failed"
	case failed == 0:
		return fmt.Sprintf("Ran %d checks, all passed", len(runs))
	case passed == 0:
		return fmt.Sprintf("Ran %d checks, all failed", len(runs))
	}

	return fmt.Sprintf("Ran %d checks: %d passed, %d failed", len(runs), passed, failed)
}

func resultText(runs []domain.RunInfo, failed bool) string {
	switch {
	case len(runs) == 0:
		return "Not verified"
	case failed:
		return "A check failed"
	}

	return "Verified"
}

// workDone reports whether the run changed files or ran a task. A run that only
// read and talked is an answer, not work to review.
func workDone(events []domain.Event) bool {
	for _, event := range events {
		if (len(event.Files) > 0 && !event.Undone) || event.Run != nil {
			return true
		}
	}

	return false
}
