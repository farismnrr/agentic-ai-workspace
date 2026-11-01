package application

import (
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// External work has no orchestrator delegation record. Its Security review must
// instead be attributed by the server to the active Security role, be newer
// than every relevant mutation, and have the evidence required by its impact.
func externalDeepReviewReady(task domain.Task, events []domain.Event) bool {
	if task.ReviewMode != domain.ReviewModeDeepReview {
		return true
	}
	impact := assessImpact(task, events)
	if !impact.security {
		return true
	}
	lastChange := int64(0)
	for _, event := range events {
		if event.Undone || !externalReviewMutation(event) {
			continue
		}
		if event.Version > lastChange {
			lastChange = event.Version
		}
	}
	// Older PASS reports cannot override a later FIX FIRST or incomplete
	// verdict. Compare global event versions, not just original sequence:
	// long-running tool events can be updated in place after they start.
	var report domain.Event
	for _, event := range events {
		if event.Version <= lastChange || event.Kind != domain.EventMessage ||
			event.Work == nil || event.Work.Role != "security" ||
			event.Work.Evidence != domain.EvidenceClaimed {
			continue
		}
		first := strings.ToLower(strings.TrimSpace(strings.SplitN(event.Text, "\n", 2)[0]))
		if strings.HasPrefix(first, "verdict:") && event.Version > report.Version {
			report = event
		}
	}
	if report.Version == 0 ||
		!strings.EqualFold(strings.TrimSpace(strings.SplitN(report.Text, "\n", 2)[0]), "Verdict: PASS") ||
		reportSection(report.Text, "Security Assessment") == "" ||
		(impact.architecture && reportSection(report.Text, "Architecture Assessment") == "") {
		return false
	}
	assessment := reportSection(report.Text, "Assessment")
	if assessment == "" || !externalReviewPathsCovered(assessment, events) ||
		!assessmentCoversCommands(assessment, events) ||
		!externalLatestChecksPass(events, lastChange) {
		return false
	}
	return !criticalSecurityImpact(events) || externalCriticalRunsPass(events, report, lastChange)
}

// Documentation edits must not invalidate a review of an unrelated code edit.
// A command with unknown effects is still an implementation mutation.
func externalReviewMutation(event domain.Event) bool {
	for _, name := range event.Files {
		if !documentationOnlyPath(name) {
			return true
		}
	}
	if event.Run != nil {
		if !event.Run.EffectsKnown || runHasImpactChange(event.Run) {
			if len(event.Run.EffectPaths) == 0 {
				return true
			}
			for _, name := range event.Run.EffectPaths {
				if !documentationOnlyPath(name) {
					return true
				}
			}
		}
	}
	return false
}

func externalReviewPathsCovered(assessment string, events []domain.Event) bool {
	for _, name := range changedPaths(events) {
		if documentationOnlyPath(name) {
			continue
		}
		if !strings.Contains(assessment, strings.ToLower(name)) {
			return false
		}
	}
	return true
}

// A retry that passes supersedes an earlier failure of the same check.
// A failed latest attempt must never be hidden by an unrelated passing run.
func externalLatestCheckAttempts(events []domain.Event, lastChange int64) map[string]domain.Event {
	latest := make(map[string]domain.Event)
	for _, event := range events {
		if event.Run == nil || event.Version <= lastChange ||
			!recognizedCheckAttempt(*event.Run) {
			continue
		}
		check := strings.TrimSpace(event.Run.Command)
		if event.Version > latest[check].Version {
			latest[check] = event
		}
	}
	return latest
}

func externalLatestChecksPass(events []domain.Event, lastChange int64) bool {
	for _, run := range externalLatestCheckAttempts(events, lastChange) {
		if !validCriticalRun(run) {
			return false
		}
	}
	return true
}

func externalCriticalRunsPass(events []domain.Event, report domain.Event, lastChange int64) bool {
	checks := uniqueChecks(declaredChecks(report.Text))
	if len(checks) == 0 {
		return false
	}
	latest := externalLatestCheckAttempts(events, lastChange)
	for _, run := range latest {
		if run.Version > report.Version || !validCriticalRun(run) {
			return false
		}
	}
	for _, check := range checks {
		run, found := latest[check]
		if !found || run.Work == nil || run.Work.Role != "security" ||
			!validCriticalRun(run) {
			return false
		}
	}
	return true
}
