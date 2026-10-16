package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func planner() application.AgentRole {
	return application.AgentRole{
		Key: "planner", Name: "Planner", Purpose: "Writes the PRD.", Tools: []string{"read_file", "ask_user"},
	}
}

func askStep(id, question string, options ...string) step {
	return call(id, "ask_user", map[string]any{"question": question, "options": options})
}

func prdReport(id, document string) step {
	return call(id, "report", map[string]string{"summary": "The PRD is ready", "document": document})
}

func TestPlannerAsksThePersonThenSavesAndSharesThePRD(t *testing.T) {
	t.Parallel()

	const prd = "# Toko kue\n## Goals\n- Orang bisa memesan kue"

	r := newTeamRig(nil, []application.AgentRole{planner(), developer()},
		delegate("d1", "planner", "Plan the bakery site"),
		askStep("q1", "Who is the site for?", "Customers (Recommended)", "The shop team"),
		prdReport("r1", prd),
		delegate("d2", "developer", "Build it"),
		report("r2", "Built"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a bakery site")
	r.run(task.ID)

	question := r.lastEvent(task.ID, domain.EventQuestion)
	if r.task(t, task.ID).Status != domain.StatusNeedsInput || question.Agent != "Planner" || len(question.Options) != 2 {
		t.Fatalf("the Planner asks the person itself, by name, with options: %+v", question)
	}

	answer := application.Answer{Choice: "Customers (Recommended)"}
	if err := r.service.Answer(context.Background(), "u1", task.ID, question.ID, answer); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	if got := resultFor(r.model.requests[2], "q1"); !strings.Contains(got.Text, "Customers (Recommended)") {
		t.Fatalf("the Planner reads the person's answer: %+v", got)
	}

	checkPRDShared(t, r, task.ID, prd)
}

// checkPRDShared checks the PRD is saved and shown, and reaches the lead and every later teammate.
func checkPRDShared(t *testing.T, r *teamRig, taskID, prd string) {
	t.Helper()

	if got := r.task(t, taskID); got.PRD != prd || r.lastEvent(taskID, domain.EventPRD).Technical != prd {
		t.Fatalf("the PRD is saved on the task and shown in the thread: %q", got.PRD)
	}

	if lead := resultFor(r.model.requests[3], "d1"); !strings.Contains(lead.Text, "Orang bisa memesan kue") {
		t.Fatalf("the lead reads the PRD too: %q", lead.Text)
	}

	brief := r.model.requests[4].Messages[0].Blocks[0].Text
	if !strings.Contains(brief, "The team's PRD:") || !strings.Contains(brief, "Orang bisa memesan kue") {
		t.Fatalf("every later teammate is shown the PRD: %q", brief)
	}
}

func TestATeammateCannotInterviewThePerson(t *testing.T) {
	t.Parallel()

	script := []step{delegate("d1", "planner", "Plan it")}
	script = append(script, askStep("q1", "One more thing?", "Yes", "No"), prdReport("r1", "# PRD"), finish("Done"))
	r := newTeamRig(nil, []application.AgentRole{planner()}, script...)
	task := r.startTeam(t, "Plan something")

	for range 6 {
		_, _ = r.mem.AppendEvent(context.Background(), task.ID, domain.Event{Kind: domain.EventQuestion, Agent: "Planner"})
	}

	r.run(task.ID)

	got := resultFor(r.model.requests[2], "q1")
	waiting := r.task(t, task.ID).Status == domain.StatusNeedsInput
	if !got.IsError || !strings.Contains(got.Text, "asked enough") || waiting {
		t.Fatalf("after six questions the Planner is told to take defaults instead of asking: %+v", got)
	}
}

func TestPromptsSayWhoAsksAndWhoDoesNot(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{planner(), researcher()},
		delegate("d1", "planner", "Plan"), prdReport("r1", "# PRD"),
		delegate("d2", "researcher", "Look"), report("r2", "Looked"), finish("Done"))
	task := r.startTeam(t, "Plan and look")
	r.run(task.ID)

	lead, planning, looking := r.model.requests[0].System, r.model.requests[1].System, r.model.requests[3].System
	wanted := []string{"asks the person its own questions", "Start building (Recommended)", "ask the person questions"}

	for _, want := range wanted {
		if !strings.Contains(lead, want) {
			t.Errorf("the lead should be told %q", want)
		}
	}

	mayAsk, cannot := strings.Contains(planning, "You may ask the person"), strings.Contains(looking, "cannot talk")
	if !mayAsk || !cannot {
		t.Error("only a teammate allowed to ask is told it may")
	}
}
