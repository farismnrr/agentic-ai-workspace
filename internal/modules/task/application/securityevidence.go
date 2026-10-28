package application

import (
	"context"
	"path"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func reviewPassed(event domain.Event) bool {
	lines := strings.Split(strings.TrimSpace(event.Technical), "\n")
	if len(lines) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(lines[0]), "Verdict: PASS")
}

func (r *Runner) hasCriticalEvidence(
	ctx context.Context, task domain.Task, events []domain.Event, report domain.Event,
) bool {
	checks := declaredChecks(report.Technical)
	if len(checks) == 0 {
		return false
	}
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, report.Job)
	if err != nil || !found {
		return false
	}
	if !assessmentCoversScope(report.Technical, events) {
		return false
	}
	lastChange := lastChangedSequence(events, d.StartSeq)
	return matchingCriticalRun(events, report, lastChange, checks)
}

func assessmentCoversScope(report string, events []domain.Event) bool {
	assessment := reportSection(report, "Assessment")
	return assessment != "" && assessmentCoversPaths(assessment, changedPaths(events)) &&
		assessmentCoversCommands(assessment, events)
}

func assessmentCoversPaths(assessment string, paths []string) bool {
	for _, name := range paths {
		if inDocuments(name) {
			continue
		}
		if !strings.Contains(assessment, strings.ToLower(path.Clean(name))) {
			return false
		}
	}
	return true
}

func assessmentCoversCommands(assessment string, events []domain.Event) bool {
	for _, event := range events {
		if event.Undone || event.Run == nil || !runHasImpactChange(event.Run) {
			continue
		}
		if strings.TrimSpace(event.Run.Command) != "" &&
			!strings.Contains(assessment, strings.ToLower(strings.TrimSpace(event.Run.Command))) {
			return false
		}
	}
	return true
}

func matchingCriticalRun(events []domain.Event, report domain.Event, lastChange int64, checks []string) bool {
	for _, check := range uniqueChecks(checks) {
		latest, ok := latestCheckRun(events, report, check, lastChange)
		if !ok || latest.Job != report.Job || !validCriticalRun(latest) {
			return false
		}
	}
	return true
}

func latestCheckRun(events []domain.Event, report domain.Event, check string, lastChange int64) (domain.Event, bool) {
	var latest domain.Event
	found := false
	for _, event := range events {
		if event.Run == nil || strings.TrimSpace(event.Run.Command) != check || event.Seq <= lastChange {
			continue
		}
		if event.Seq > report.Seq {
			return domain.Event{}, false
		}
		latest, found = event, true
	}
	return latest, found
}

func validCriticalRun(event domain.Event) bool {
	return event.Run != nil && event.Run.EffectsKnown && len(event.Run.EffectPaths) == 0 &&
		passedRun(*event.Run) && meaningfulCheck(*event.Run)
}

func lastChangedSequence(events []domain.Event, after int64) int64 {
	var last int64
	for _, event := range events {
		if event.Seq > after && !event.Undone && hasImpactChange(event) && event.Seq > last {
			last = event.Seq
		}
	}
	return last
}

func inDocuments(name string) bool { return strings.HasPrefix(path.Clean(name), "documents/") }

func hasImpactChange(event domain.Event) bool {
	if hasChangedFiles(event.Files) {
		return true
	}
	return runHasImpactChange(event.Run)
}

func runHasImpactChange(run *domain.RunInfo) bool {
	return run != nil && (!run.EffectsKnown || len(run.EffectPaths) > 0 || !knownNonMutatingCommand(run.Command))
}

func hasChangedFiles(files []string) bool {
	for _, name := range files {
		if !inDocuments(name) {
			return true
		}
	}
	return false
}
