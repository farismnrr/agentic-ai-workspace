package application

import (
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// External work has no orchestrator delegation record. Its Security review must
// ExternalDeepReviewReady verifies if external deep review has all required evidence.
func ExternalDeepReviewReady(task domain.Task, events []domain.Event) bool {
	return ExternalDeepReviewMissing(task, events) == ""
}

func externalDeepReviewReady(task domain.Task, events []domain.Event) bool {
	return ExternalDeepReviewReady(task, events)
}

func externalDeepReviewMissing(task domain.Task, events []domain.Event) string {
	return ExternalDeepReviewMissing(task, events)
}

// ExternalDeepReviewMissing reports any missing requirements for deep review.
func ExternalDeepReviewMissing(task domain.Task, events []domain.Event) string {
	if task.ReviewMode != domain.ReviewModeDeepReview {
		return ""
	}
	impact := assessImpact(task, events)
	if !impact.security {
		return ""
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
		!strings.EqualFold(strings.TrimSpace(strings.SplitN(report.Text, "\n", 2)[0]), "Verdict: PASS") {
		return "fresh Security PASS verdict missing"
	}
	if reportSection(report.Text, "Security Assessment") == "" ||
		(impact.architecture && reportSection(report.Text, "Architecture Assessment") == "") {
		return "Security or Architecture Assessment missing"
	}
	assessment := reportSection(report.Text, "Assessment")
	if assessment == "" || !externalReviewPathsCovered(assessment, events) {
		return "Assessment must cover changed implementation paths"
	}
	if !externalReviewCommandsCovered(report.Text, events) {
		return "command audit incomplete; cover impacted commands or provide explicit Command Audit"
	}
	if !externalLatestChecksPass(events, lastChange) {
		return "latest recorded check failed or has unverified effects"
	}
	if criticalSecurityImpact(events) && !externalCriticalRunsPass(events, report, lastChange) {
		return "matching successful Security-run critical check missing"
	}
	return ""
}

// Security can enumerate individual commands in Assessment. For long-lived
// work sessions this may exceed the 2,000-character progress report limit.
// An explicit attestation of ALL recorded command activity is an alternative,
// but never substitutes for changed-path coverage, a fresh Security verdict,
// or a real server-recorded critical check after the last mutation.
func externalReviewCommandsCovered(report string, events []domain.Event) bool {
	if assessmentCoversCommands(reportSection(report, "Assessment"), events) {
		return true
	}
	audit := reportSection(report, "Command Audit")
	return strings.Contains(audit, "reviewed all recorded task commands, including unverified workspace effects")
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
