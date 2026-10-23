package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var errTeamStoreOffline = errors.New("team store offline")

func TestCreateRejectsAnUnavailableTeamBeforeSaving(t *testing.T) {
	tests := []struct {
		team *fakeTeam
		name string
	}{
		{name: "roster error", team: &fakeTeam{err: errTeamStoreOffline}},
		{name: "missing lead", team: &fakeTeam{roster: application.Roster{Members: []application.AgentRole{developer()}}}},
		{name: "no active teammate", team: &fakeTeam{roster: application.Roster{
			Lead: application.AgentRole{Key: "lead", Name: "Lead", Lead: true},
		}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
				deps.Team = test.team
				deps.Scopes, deps.Delegations = deps.Transcript.(*memory), newDelegations()
			})
			yes := true
			_, err := r.service.Create(context.Background(), "u1", application.CreateInput{
				ProjectID: "proj", Prompt: "Use the team", Team: &yes,
			})
			if !errors.Is(err, domain.ErrTeamUnavailable) || len(r.mem.tasks) != 0 || len(r.disp.enqueued) != 0 {
				t.Fatalf("unavailable team must be rejected before task creation: err=%v tasks=%d queued=%v",
					err, len(r.mem.tasks), r.disp.enqueued)
			}
		})
	}
}

func TestExplicitTeamModeNeedsAllTeamDependencies(t *testing.T) {
	r := newRig(nil)
	yes := true
	_, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: "Use the team", Team: &yes,
	})
	if !errors.Is(err, domain.ErrTeamUnavailable) {
		t.Fatalf("missing team dependencies must reject Orchestrator mode: %v", err)
	}

	no := false
	task, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: "Work directly", Team: &no,
	})
	if err != nil || task.Team {
		t.Fatalf("Solo stays available without team dependencies: %+v %v", task, err)
	}
}

func TestLegacyTeamModeSurvivesFollowUpAndRetry(t *testing.T) {
	for _, action := range []string{"message", "retry"} {
		t.Run(action, func(t *testing.T) { assertLegacyTeamModeSurvives(t, action) })
	}
}

func assertLegacyTeamModeSurvives(t *testing.T, action string) {
	t.Helper()
	r := newTeamRig(nil, []application.AgentRole{developer()}, finish("Continued."))
	legacy := domain.Task{
		ID: "legacy", ProjectID: "proj", UserID: "u1", ProviderID: "prov", Model: "model-x",
		Team: true, Status: domain.StatusFailed, Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	if err := continueLegacyTask(r, legacy.ID, action); err != nil {
		t.Fatal(err)
	}
	assertLegacyTaskQueued(t, r, legacy.ID)
	r.run(legacy.ID)
	assertLegacyTaskFinished(t, r, legacy.ID)
}

func continueLegacyTask(r *teamRig, taskID, action string) error {
	if action == "message" {
		err := r.service.FollowUp(context.Background(), "u1", taskID, "Keep using the team")
		return wrapTaskActionError("follow up", err)
	}
	err := r.service.Retry(context.Background(), "u1", taskID)
	return wrapTaskActionError("retry", err)
}

func wrapTaskActionError(action string, err error) error {
	if err != nil {
		return fmt.Errorf("%s legacy team task: %w", action, err)
	}
	return nil
}

func assertLegacyTaskQueued(t *testing.T, r *teamRig, taskID string) {
	t.Helper()
	if got := r.task(t, taskID); !got.Team || got.Status != domain.StatusQueued {
		t.Fatalf("continuation preserves the stored team mode: %+v", got)
	}
}

func assertLegacyTaskFinished(t *testing.T, r *teamRig, taskID string) {
	t.Helper()
	if got := r.task(t, taskID); !got.Team || got.Status != domain.StatusDone {
		t.Fatalf("legacy task resumes through the team runner: %+v", got)
	}
	if !hasAny(toolNames(r.model.requests[0]), "delegate") {
		t.Fatal("the continued task keeps Orchestrator tools")
	}
}
