package application_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestSwitchingBetweenAPIAndVendorProgramsKeepsContextAndSeparatesSessions(t *testing.T) {
	t.Parallel()
	team := &fakeTeam{roster: application.Roster{
		Lead:    application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true},
		Members: []application.AgentRole{developer()},
	}}
	program, sessions := &scriptedProgram{say: "CLI-ANSWER-SENTINEL"}, sessionBook{}
	r := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		deps.Team, deps.Scopes, deps.Delegations = team, deps.Transcript.(*memory), newDelegations()
		deps.CLI, deps.Sessions = program, sessions
	}, finish("API-ANSWER-SENTINEL"), finish("API-RETURN-SENTINEL"))
	task := r.start(t, "Remember this task")
	r.run(task.ID)
	team.roster.Lead.ProviderID, team.roster.Lead.Model = "plan:claude", "sonnet"
	if err := r.service.FollowUp(t.Context(), "u1", task.ID, "Continue in the program"); err != nil {
		t.Fatal(err)
	}
	r.run(task.ID)
	if len(program.requests) != 1 {
		t.Fatal("settings did not switch the task to the vendor program")
	}
	req := program.requests[0]
	if req.Session != "" || !strings.Contains(req.Prompt, "API-ANSWER-SENTINEL") {
		t.Fatalf("the fresh program needs prior task context: %+v", req)
	}
	team.roster.Lead.ProviderID, team.roster.Lead.Model = "prov", "new-api-model"
	if err := r.service.FollowUp(t.Context(), "u1", task.ID, "Continue through API"); err != nil {
		t.Fatal(err)
	}
	r.run(task.ID)
	if sessions[task.ID] != "" || r.task(t, task.ID).Status != domain.StatusDone {
		t.Fatal("session was reused across providers")
	}
	last := r.model.requests[len(r.model.requests)-1]
	content, _ := json.Marshal(last.Messages)
	if last.Model != "new-api-model" || !strings.Contains(string(content), "CLI-ANSWER-SENTINEL") {
		t.Fatalf("returning to API lost the program's task context: %s", content)
	}
}
