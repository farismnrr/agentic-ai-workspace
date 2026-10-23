package application_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestQueuedTeamTaskFailsClosedAndCanRetryAfterRecovery(t *testing.T) {
	team := readyTeam()
	r := teamRunRig(team, finish("The team continued."))
	task := createTeamTask(t, r)
	if err := r.delegations.StartDelegation(context.Background(), task.ID, application.Delegation{
		ID: "d1", RoleKey: "developer", Status: application.DelegationWorking,
	}, task.CreatedAt); err != nil {
		t.Fatal(err)
	}
	before, err := r.mem.LoadTranscript(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	team.err = errTeamStoreOffline
	r.run(task.ID)
	assertUnavailableFailure(t, r, task.ID)
	after, err := r.mem.LoadTranscript(context.Background(), task.ID)
	if err != nil || !reflect.DeepEqual(before, after) || r.delegations.status(task.ID) != application.DelegationWorking {
		t.Fatalf("failed readiness check keeps transcript and pending teammate job: err=%v transcript=%v job=%q",
			err, reflect.DeepEqual(before, after), r.delegations.status(task.ID))
	}

	team.err = nil
	if err := r.service.Retry(context.Background(), "u1", task.ID); err != nil {
		t.Fatal(err)
	}
	if !r.task(t, task.ID).Team {
		t.Fatal("Retry must preserve the saved Orchestrator mode")
	}
	r.run(task.ID)
	if got := r.task(t, task.ID); got.Status != domain.StatusDone || !got.Team {
		t.Fatalf("the same task resumes with its team after recovery: %+v", got)
	}
}

func TestQueuedTeamTaskFailsClosedForCLIProvider(t *testing.T) {
	team := readyTeam()
	r := teamRunRig(team)
	task := createTeamTask(t, r)
	stored := r.mem.tasks[task.ID]
	stored.ProviderID, stored.FollowSettings = "plan:chatgpt", false
	r.mem.tasks[task.ID] = stored
	team.err = errTeamStoreOffline

	r.run(task.ID)
	assertUnavailableFailure(t, r, task.ID)
	if len(r.model.requests) != 0 {
		t.Fatalf("a CLI task with no team must stop before calling any model: %d calls", len(r.model.requests))
	}
}

func readyTeam() *fakeTeam {
	return &fakeTeam{roster: application.Roster{
		Lead:    application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true},
		Members: []application.AgentRole{developer()},
	}}
}

func teamRunRig(team *fakeTeam, steps ...step) *teamRunHarness {
	delegations := newDelegations()
	rig := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		deps.Team, deps.Scopes, deps.Delegations = team, deps.Transcript.(*memory), delegations
	}, steps...)
	return &teamRunHarness{rig: rig, team: team, delegations: delegations}
}

type teamRunHarness struct {
	*rig
	team        *fakeTeam
	delegations *fakeDelegations
}

func createTeamTask(t *testing.T, r *teamRunHarness) domain.Task {
	t.Helper()
	yes := true
	task, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: "Keep the current work", Team: &yes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func assertUnavailableFailure(t *testing.T, r *teamRunHarness, taskID string) {
	t.Helper()
	task := r.task(t, taskID)
	if task.Status != domain.StatusFailed || !task.Team {
		t.Fatalf("task keeps its Orchestrator mode and fails for recovery: %+v", task)
	}
	events, _ := r.mem.EventsAfter(context.Background(), taskID, 0)
	last := events[len(events)-1]
	if last.Kind != domain.EventError || !strings.Contains(last.Hint, "Retry or Continue") {
		t.Fatalf("failure explains how to recover the team task: %+v", last)
	}
}
