package task_test

import (
	"strings"
	"testing"

	teamdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// P0 lifecycle invariant: a correct cross-role handoff is not a failed job.
func TestRoleDependencyRemainsPending(t *testing.T) {
	for _, tc := range []struct {
		name string
		report string
		failed bool
		state taskdomain.PlanState
	}{
		{name: "DevOps prerequisite", report: "Needs DevOps: start_site before QA browser_test", state: taskdomain.PlanPending},
		{name: "legacy app prerequisite", report: "Needs the app started", state: taskdomain.PlanPending},
		{name: "QA prerequisite", report: "Needs QA: browser_test", state: taskdomain.PlanPending},
		{name: "research prerequisite", report: "Needs research: official API docs", state: taskdomain.PlanPending},
		{name: "actual error", report: "Stuck: build failed twice", state: taskdomain.PlanFailed},
		{name: "normal completion", report: "Completed source audit with evidence.", state: taskdomain.PlanDone},
		{name: "failed worker", report: "Needs DevOps: start_site", failed: true, state: taskdomain.PlanFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, _ := taskapp.PlanStateAfterForTest(tc.failed, tc.report)
			if state != tc.state {
				t.Fatalf("report %q: got %s, want %s", tc.report, state, tc.state)
			}
		})
	}
}

// Read-only work must not count as an endless series of useless delegations.
func TestSuccessfulReadOnlyJobResetsIdleCounter(t *testing.T) {
	events := []taskdomain.Event{
		{Kind: taskdomain.EventHandoff, Action: "assigned"},
		{Kind: taskdomain.EventHandoff, Action: "reported", Technical: "Read source and produced an evidenced audit."},
		{Kind: taskdomain.EventHandoff, Action: "assigned"},
	}
	if idle := taskapp.IdleJobsForTest(events); idle != 1 {
		t.Fatalf("read-only result should reset the counter: got %d, want 1", idle)
	}
	events = []taskdomain.Event{
		{Kind: taskdomain.EventHandoff, Action: "assigned"},
		{Kind: taskdomain.EventHandoff, Action: "reported", Technical: "Stuck: missing tool"},
		{Kind: taskdomain.EventHandoff, Action: "assigned"},
	}
	if idle := taskapp.IdleJobsForTest(events); idle != 2 {
		t.Fatalf("blocked report must not reset stuck counter: got %d, want 2", idle)
	}
}

func TestHistoricRoleDefaultsUpgradeWithoutOverwritingCustomText(t *testing.T) {
	team := teamdomain.Presets()
	for i := range team.Roles {
		switch team.Roles[i].Key {
		case "developer":
			team.Roles[i].Purpose = "Writes and changes the code, following the design."
			team.Roles[i].Prompt = "You are a careful developer. Look before you change, make small focused edits that match the " +
				"surrounding code, write it with SOLID and Clean Architecture, and run the relevant build or tests " +
				"after your last change. Report what you changed and what you ran."
		case "qa":
			team.Roles[i].Name = "QA Tester"
			team.Roles[i].Purpose = "Checks the work by running the tests and trying to break it."
			team.Roles[i].Prompt = "Custom owner instruction: review carefully."
		}
	}
	upgraded := team.WithBuiltIns()
	for _, role := range upgraded.Roles {
		switch role.Key {
		case "developer":
			if strings.Contains(role.Prompt, "run the relevant build or tests") || !strings.Contains(role.Prompt, "Do not run QA tests") {
				t.Fatalf("stale shipped Developer prompt survived migration: %q", role.Prompt)
			}
		case "qa":
			if role.Name != "QA & Review" || role.Prompt != "Custom owner instruction: review carefully." {
				t.Fatalf("QA must upgrade known name but preserve custom prompt: %+v", role)
			}
			if !strings.Contains(role.Purpose, "independently reviews") {
				t.Fatalf("old shipped QA purpose was not upgraded: %q", role.Purpose)
			}
		}
	}
}
