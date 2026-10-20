package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func askFirst(lim *application.Limits) { lim.AskFirst = true }

func TestTheOrchestratorDecidesWhetherToAskBeforeTheFirstJobOfARequest(t *testing.T) {
	t.Parallel()

	r := newTeamRig(askFirst, []application.AgentRole{buildingDeveloper()},
		delegate("d1", "developer", "Make the page"),
		delegate("d2", "developer", "Make the page"),
		report("r1", "Made"),
		finish("Done"),
	)
	r.gate.skip = true
	task := r.startTeam(t, "Make a page")
	r.run(task.ID)

	held := resultFor(r.model.requests[1], "d1")
	if !held.IsError || !strings.Contains(held.Text, "ask_user") || !strings.Contains(held.Text, "(Recommended)") {
		t.Fatalf("the first job waits while the orchestrator decides whether to ask: %+v", held)
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("a clear request goes on once the orchestrator sends the job again: %q", got)
	}
}

func TestAQuestionAlreadyAskedSinceTheRequestIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	r := newTeamRig(askFirst, []application.AgentRole{buildingDeveloper()},
		delegate("d1", "developer", "Make the page"),
		report("r1", "Made"),
		finish("Done"),
	)
	r.gate.skip = true
	task := r.startTeam(t, "Make a page")

	asked := domain.Event{Kind: domain.EventQuestion, Text: "Which colour?"}
	if _, err := r.mem.AppendEvent(t.Context(), task.ID, asked); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	if held := resultFor(r.model.requests[1], "d1"); held.IsError {
		t.Fatalf("the person was asked already, the job starts: %+v", held)
	}
}
