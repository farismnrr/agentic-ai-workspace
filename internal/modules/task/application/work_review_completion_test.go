package application

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestExternalDeepReviewLongHistoryRequiresExplicitCommandAuditAndRealCheck(t *testing.T) {
	const cmd = "go test -count=1 ./internal/modules/task/..."
	task := domain.Task{ReviewMode: domain.ReviewModeDeepReview}
	events := []domain.Event{
		{Kind: domain.EventAction, Version: 1, Files: []string{"internal/modules/task/application/example.go"}},
		{Kind: domain.EventAction, Version: 2, Run: &domain.RunInfo{
			Name: "git", Command: "git -c core.hooksPath=/dev/null --no-pager commit -m patch",
			EffectsKnown: false,
		}},
		{Kind: domain.EventAction, Version: 3, Work: &domain.WorkRef{Role: "security", Evidence: domain.EvidenceRan},
			Run: &domain.RunInfo{Name: "command", Command: cmd, EffectsKnown: true, TestExecuted: true}},
		{Kind: domain.EventMessage, Version: 4,
			Work: &domain.WorkRef{Role: "security", Evidence: domain.EvidenceClaimed},
		},
	}
	report := "Verdict: PASS\n" +
		"Security Assessment: Sensitive implementation and historical Git activity reviewed.\n" +
		"Architecture Assessment: API contracts and failed snapshots reviewed.\n" +
		"Assessment: internal/modules/task/application/example.go\n" +
		"Checks:\n- " + cmd
	events[3].Text = report
	if externalDeepReviewReady(task, events) {
		t.Fatal("historical unknown-effect command may not be omitted silently")
	}
	events[3].Text = "Verdict: PASS\n" +
		"Security Assessment: Sensitive implementation reviewed.\n" +
		"Architecture Assessment: Historical command effects reviewed.\n" +
		"Assessment: internal/modules/task/application/example.go\n" +
		"Command Audit: reviewed all recorded task commands, including unverified workspace effects\n" +
		"Checks:\n- " + cmd
	if missing := externalDeepReviewMissing(task, events); missing != "" {
		t.Fatalf("explicit scoped audit and genuine check should permit completion: %s", missing)
	}
	events[2].Run.EffectsKnown = false
	if externalDeepReviewReady(task, events) {
		t.Fatal("declared command audit cannot replace observed safe check effects")
	}
	events[2].Run.EffectsKnown = true
	events[2].Run.TestExecuted = false
	if externalDeepReviewReady(task, events) {
		t.Fatal("declared command audit cannot replace executed test evidence")
	}
	events[2].Run.TestExecuted = true
	events[3].Work.Role = "developer"
	if externalDeepReviewReady(task, events) {
		t.Fatal("developer cannot impersonate Security verdict")
	}
	events[3].Work.Role = "security"
	events = append(events, domain.Event{Kind: domain.EventAction, Version: 5,
		Files: []string{"internal/modules/task/application/example.go"}})
	if externalDeepReviewReady(task, events) {
		t.Fatal("subsequent code changes must invalidate the earlier audit and check")
	}
}
