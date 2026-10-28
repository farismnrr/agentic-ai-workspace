package application

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// buildProof derives the receipt from every recognized check attempt after the
// latest change. A failed current attempt stays visible after an earlier pass.
func buildProof(events []domain.Event, claims []string) *domain.Proof {
	runs, by := runsSinceLastChange(events)
	proof := &domain.Proof{Checks: []domain.Check{}}
	failed := false
	verified := len(runs) > 0

	for i, run := range runs {
		passed := passedRun(run)
		failed = failed || !passed
		verified = verified && verificationRun(run)
		check := domain.Check{Text: runText(run, passed), Verified: true, Passed: passed, By: by[i]}
		proof.Checks = append(proof.Checks, check)
	}

	for _, claim := range claims {
		proof.Checks = append(proof.Checks, domain.Check{Text: claim})
	}

	proof.Proven = verified && !failed
	proof.Rows = []domain.ProofRow{
		{Label: "What changed", Value: changedSummary(events)},
		{Label: "How it was tested", Value: testedSummary(runs)},
		{Label: "Result", Value: resultText(runs, failed, proof.Proven)},
	}

	return proof
}

// runsSinceLastChange are the checks the server ran after the last change, and who ran each one.
func runsSinceLastChange(events []domain.Event) ([]domain.RunInfo, []string) {
	start := 0

	for i, event := range events {
		if !event.Undone && hasImpactChange(event) {
			if event.Run != nil && recognizedCheckAttempt(*event.Run) {
				continue
			}
			start = i + 1
		}
	}

	runs, by := []domain.RunInfo{}, []string{}

	for _, event := range events[start:] {
		if event.Run != nil && recognizedCheckAttempt(*event.Run) {
			runs, by = append(runs, *event.Run), append(by, event.Agent)
		}
	}

	return runs, by
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
	files := changedFiles(events)
	summary := fileCountSummary(len(files))
	if !hasUnknownEffects(events) {
		return summary
	}
	if len(files) == 0 {
		return "File changes could not be confirmed"
	}
	return summary + "; other file changes could not be confirmed"
}

func changedFiles(events []domain.Event) map[string]bool {
	files := map[string]bool{}
	for _, event := range events {
		if event.Undone {
			continue
		}

		for _, file := range event.Files {
			files[file] = true
		}
		if event.Run != nil {
			for _, file := range event.Run.EffectPaths {
				files[file] = true
			}
		}
	}
	return files
}

func hasUnknownEffects(events []domain.Event) bool {
	for _, event := range events {
		if event.Undone || event.Run == nil {
			continue
		}
		if !event.Run.EffectsKnown || len(event.Run.EffectPaths) == 0 && !knownNonMutatingCommand(event.Run.Command) {
			return true
		}
	}
	return false
}

func fileCountSummary(count int) string {
	switch count {
	case 0:
		return "No files were changed"
	case 1:
		return "1 file changed"
	}
	return fmt.Sprintf("%d files changed", count)
}

func recognizedCheckAttempt(run domain.RunInfo) bool {
	return knownCheckCommand(run.Command) && !noOpCheckCommand(run.Command) &&
		(!isGoTestCommand(run.Command) || GoTestExecutionEvidenceAllowed(run.Command))
}

func verificationRun(run domain.RunInfo) bool {
	return run.EffectsKnown && len(run.EffectPaths) == 0 && meaningfulCheck(run)
}

// passedRun reports whether a check the server ran succeeded.
func passedRun(run domain.RunInfo) bool {
	return run.ExitCode == 0 && !run.TimedOut && meaningfulCheck(run)
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

func resultText(runs []domain.RunInfo, failed, proven bool) string {
	switch {
	case len(runs) == 0:
		return "Not verified"
	case failed:
		return "A check failed"
	}

	if !proven {
		return "Not verified"
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
