package application

import (
	"context"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"strings"
	"testing"
	"time"
)

type cliReviewTasks struct {
	TaskStore
	task      domain.Task
	completed bool
}

func (s *cliReviewTasks) Find(context.Context, string, string) (domain.Task, error) {
	return s.task, nil
}
func (*cliReviewTasks) SetContext(context.Context, string, int64, int64) error { return nil }
func (*cliReviewTasks) SetContextState(context.Context, string, string) error  { return nil }
func (s *cliReviewTasks) Finish(context.Context, string, string, *domain.Proof, time.Time) error {
	s.completed = true
	return nil
}

type cliReviewEvents struct {
	EventStore
	entries []domain.Event
}

func (s *cliReviewEvents) EventsAfter(_ context.Context, _ string, seq int64) ([]domain.Event, error) {
	out := []domain.Event{}
	for _, e := range s.entries {
		if e.Seq > seq {
			out = append(out, e)
		}
	}
	return out, nil
}
func (s *cliReviewEvents) AppendEvent(_ context.Context, _ string, e domain.Event) (domain.Event, error) {
	e.Seq = int64(len(s.entries) + 1)
	s.entries = append(s.entries, e)
	return e, nil
}

type cliReviewTeam struct{ roster Roster }

func (s cliReviewTeam) Roster(context.Context, string) (Roster, error) { return s.roster, nil }

type cliReviewScopes struct{ ScopedTranscripts }
type cliReviewDelegations struct {
	DelegationStore
	reported bool
}

func (s *cliReviewDelegations) FindDelegation(_ context.Context, _, id string) (Delegation, bool, error) {
	if s.reported && id == "review-job" {
		return Delegation{ID: id, RoleKey: "reviewer", RoleName: "Reviewer", Status: DelegationDone}, true, nil
	}
	return Delegation{}, false, nil
}

type cliReviewGate struct{ ToolGate }

func (cliReviewGate) Enabled(context.Context, string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

type cliReviewProjects struct{ Projects }

func (cliReviewProjects) Name(context.Context, string, string) (string, error) {
	return "Existing project", nil
}

type cliReviewNotifier struct{}

func (cliReviewNotifier) Changed(string, string) {}
func (cliReviewNotifier) Delta(string, string)   {}
func (cliReviewNotifier) FilesChanged(string)    {}

type cliReviewSessions struct{ session string }

func (s *cliReviewSessions) Session(context.Context, string) (string, error) { return s.session, nil }
func (s *cliReviewSessions) SaveSession(_ context.Context, _, value string) error {
	s.session = value
	return nil
}

type cliReviewVendor struct {
	requests      []CLIRequest
	events        *cliReviewEvents
	delegations   *cliReviewDelegations
	publishReport bool
	noSession     bool
	skipChange    bool
}

func (*cliReviewVendor) Ready(context.Context, string, string) (bool, error) { return true, nil }
func (s *cliReviewVendor) Run(ctx context.Context, req CLIRequest, emit func(CLIEvent)) (CLIResult, error) {
	s.requests = append(s.requests, req)
	if len(s.requests) == 1 && !s.skipChange {
		emit(CLIEvent{Kind: CLIChange, Text: "Changed label", Files: []string{"web/src/label.css"}})
	} else if len(s.requests) > 1 && s.publishReport {
		s.delegations.reported = true
		_, _ = s.events.AppendEvent(ctx, req.TaskID, domain.Event{
			Kind: domain.EventHandoff, Action: handoffReported, Job: "review-job", Agent: "Reviewer",
			Technical: "Verdict: PASS\nChanged CSS meets requirements", Text: "Reviewer completed independent review",
		})
	}
	if s.noSession {
		return CLIResult{Summary: "Complete"}, nil
	}
	return CLIResult{Session: "vendor-session", Summary: "Complete"}, nil
}
func newCLIReviewHarness(t *testing.T) (*Runner, domain.Task, *cliReviewVendor, *cliReviewTasks, *cliReviewEvents) {
	t.Helper()
	task := domain.Task{ID: "review-task", UserID: "owner", ProjectID: "project", Team: true,
		ReviewMode: domain.ReviewModeDeepReview, ProviderID: "plan:chatgpt"}
	tasks := &cliReviewTasks{task: task}
	events := &cliReviewEvents{}
	_, _ = events.AppendEvent(context.Background(), task.ID, domain.Event{Kind: domain.EventRequest, Text: "Update the label"})
	delegations := &cliReviewDelegations{}
	vendor := &cliReviewVendor{events: events, delegations: delegations}
	team := cliReviewTeam{roster: Roster{
		Lead:    AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true},
		Members: []AgentRole{{Key: "reviewer", Name: "Reviewer", ChecksLook: true}},
	}}
	r := NewRunner(Deps{
		Tasks: tasks, Events: events, CLI: vendor, Sessions: &cliReviewSessions{session: "old-session"},
		Team: team, Scopes: cliReviewScopes{}, Delegations: delegations,
		Gate: cliReviewGate{}, Projects: cliReviewProjects{}, Notify: cliReviewNotifier{}, Now: time.Now,
	}, DefaultLimits())
	return r, task, vendor, tasks, events
}

func TestCLIReviewAutoContinuesWithinSameTaskWhenReviewerReports(t *testing.T) {
	r, task, vendor, tasks, events := newCLIReviewHarness(t)
	vendor.publishReport = true
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFinished || !tasks.completed {
		t.Fatalf("valid review must allow completion: %+v completed=%v", result, tasks.completed)
	}
	if len(vendor.requests) != 2 {
		t.Fatalf("expected one automatic review follow-up, got %d", len(vendor.requests))
	}
	if vendor.requests[0].TaskID != vendor.requests[1].TaskID || vendor.requests[1].Session != "vendor-session" {
		t.Fatalf("review restarted or split task: %+v", vendor.requests)
	}
	if !strings.Contains(vendor.requests[1].Prompt, "Required verification:") ||
		!strings.Contains(vendor.requests[1].Prompt, "Reviewer") || vendor.requests[1].Bridge == nil {
		t.Fatalf("CLI did not receive actionable reviewer feedback: %+v", vendor.requests[1])
	}
	for _, e := range events.entries {
		if e.Kind == domain.EventError {
			t.Fatalf("successful review caused error card: %+v", e)
		}
	}
}

func TestCLIReviewMissingReportStopsWithSpecificReasonWithoutRetryCards(t *testing.T) {
	r, task, vendor, tasks, events := newCLIReviewHarness(t)
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFailed || !strings.Contains(result.text, "Reviewer review is still missing") {
		t.Fatalf("should fail with specific mandatory review evidence reason: %+v", result)
	}
	if tasks.completed || len(vendor.requests) <= 1 || len(vendor.requests) > maxCLIReviewContinuations+1 {
		t.Fatalf("missing review silently passed or ran forever: completed=%v calls=%d", tasks.completed, len(vendor.requests))
	}
	for _, e := range events.entries {
		if e.Kind == domain.EventError {
			t.Fatalf("auto-review stored duplicate error card: %+v", e)
		}
	}
}

func TestCLIReviewDisabledReviewerReportsActualBlocker(t *testing.T) {
	r, task, vendor, tasks, _ := newCLIReviewHarness(t)
	team := r.deps.Team.(cliReviewTeam)
	team.roster.Members = []AgentRole{{Key: "developer", Name: "Developer"}}
	r.deps.Team = team
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFailed || !strings.Contains(result.text, "no matching role is enabled") || tasks.completed || len(vendor.requests) != 1 {
		t.Fatalf("missing reviewer must block with useful error: %+v calls=%d", result, len(vendor.requests))
	}
}

func TestCLIKnownReadOnlyCommandDoesNotInventUnknownEffects(t *testing.T) {
	r, task, _, _, events := newCLIReviewHarness(t)
	sink := &planSink{runner: r, task: task}
	sink.emit(context.Background(), CLIEvent{Kind: CLICommand, Text: "Searched repository", Technical: "rg --files -g 'AGENTS.md' -g '*_test.go'"})
	got := events.entries[len(events.entries)-1]
	if got.Run == nil || !got.Run.EffectsKnown || len(got.Run.EffectPaths) != 0 {
		t.Fatalf("read-only rg invented unrecorded file modifications: %+v", got)
	}
	impact := assessImpact(task, events.entries)
	if impact.unknown || impact.security || impact.reviewer || impact.architecture {
		t.Fatalf("read-only rg incorrectly triggered mandatory reviews: %+v", impact)
	}
}
func TestCLIUnknownCommandRetainsConservativeReview(t *testing.T) {
	r, task, _, _, events := newCLIReviewHarness(t)
	sink := &planSink{runner: r, task: task}
	sink.emit(context.Background(), CLIEvent{Kind: CLICommand, Text: "Ran shell command", Technical: "sh -c 'touch src/config.go'"})
	got := events.entries[len(events.entries)-1]
	if got.Run == nil || got.Run.EffectsKnown {
		t.Fatalf("unclassified command was incorrectly declared safe: %+v", got)
	}
	impact := assessImpact(task, events.entries)
	if !impact.unknown || !impact.security || !impact.reviewer || !impact.architecture {
		t.Fatalf("unclassified effects did not trigger conservative reviews: %+v", impact)
	}
}

func TestCLIReviewContinuesExistingTaskWithOldBlockedCards(t *testing.T) {
	r, task, vendor, tasks, events := newCLIReviewHarness(t)
	vendor.skipChange = true
	vendor.publishReport = true
	_, _ = events.AppendEvent(context.Background(), task.ID, domain.Event{Kind: domain.EventAction, Text: "Earlier Developer changed label", Files: []string{"web/src/label.css"}})
	for range 3 {
		_, _ = events.AppendEvent(context.Background(), task.ID, domain.Event{Kind: domain.EventError, Text: "This work is blocked until its required review is recorded."})
	}
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFinished || !tasks.completed || len(vendor.requests) != 2 {
		t.Fatalf("Continue did not recover old blocked task: %+v completed=%v calls=%d", result, tasks.completed, len(vendor.requests))
	}
	errors := 0
	for _, e := range events.entries {
		if e.Kind == domain.EventError {
			errors++
		}
	}
	if errors != 3 {
		t.Fatalf("added another generic error card on resumed task, total=%d", errors)
	}
}
func TestCLISwiftChangeDoesNotForceIndependentReviewer(t *testing.T) {
	r, task, vendor, tasks, _ := newCLIReviewHarness(t)
	task.ReviewMode = domain.ReviewModeSwift
	tasks.task = task
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFinished || !tasks.completed || len(vendor.requests) != 1 {
		t.Fatalf("Swift ordinary CSS work demanded a review: %+v calls=%d", result, len(vendor.requests))
	}
}
func TestCLIReviewCannotLoopWithoutResumableVendorSession(t *testing.T) {
	r, task, vendor, tasks, _ := newCLIReviewHarness(t)
	vendor.noSession = true
	r.deps.Sessions = &cliReviewSessions{}
	result := r.turnOnPlan(context.Background(), task, "chatgpt")
	if result.stop != stopFailed || !strings.Contains(result.text, "resumable session") || len(vendor.requests) != 1 || tasks.completed {
		t.Fatalf("cannot safely auto-resume anonymous vendor: %+v calls=%d", result, len(vendor.requests))
	}
}
