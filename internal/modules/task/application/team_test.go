package application_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func toolNames(request application.Request) []string {
	names := []string{}
	for _, tool := range request.Tools {
		names = append(names, tool.Name)
	}

	return names
}

func resultFor(request application.Request, id string) domain.Block {
	for _, message := range request.Messages {
		for _, block := range message.Blocks {
			if block.Kind == domain.BlockToolResult && block.ID == id {
				return block
			}
		}
	}

	return domain.Block{}
}

func TestDelegatedJobRunsOnTheTeammatesModelAndReportsBack(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Fix the button"),
		call("c1", "read_file", map[string]string{"path": "a.txt"}),
		report("r1", "The button is fixed"),
		finish("All done"),
	)
	task := r.startTeam(t, "Fix the button")
	r.run(task.ID)

	if got := r.task(t, task.ID); got.Status != domain.StatusDone || !got.Team {
		t.Fatalf("a team task that changed nothing is answered like any other: %+v", got)
	}

	if r.delegations.status(task.ID) != application.DelegationDone {
		t.Fatalf("the delegation must end as done: %q", r.delegations.status(task.ID))
	}

	requests := r.model.requests
	if requests[0].Model != "model-x" || requests[1].Model != "dev-model" || requests[3].Model != "model-x" {
		t.Fatalf("the teammate uses its own model, the orchestrator the task's: %q %q %q",
			requests[0].Model, requests[1].Model, requests[3].Model)
	}

	back := resultFor(requests[3], "d1").Text
	if !strings.Contains(back, "The button is fixed") || !strings.Contains(back, "recorded for Developer") {
		t.Fatalf("the orchestrator reads the report plus what the server recorded: %q", back)
	}
}

// hasAny reports whether names holds any of want.
func hasAny(names []string, want ...string) bool {
	return slices.ContainsFunc(want, func(name string) bool { return slices.Contains(names, name) })
}

// stepsBy counts the actions a teammate did and lists how the handoffs went.
func stepsBy(events []domain.Event, agent string) (int, []string) {
	steps, handoffs := 0, []string{}

	for _, event := range events {
		switch {
		case event.Kind == domain.EventAction && event.Agent == agent:
			steps++
		case event.Kind == domain.EventHandoff:
			handoffs = append(handoffs, event.Action)
		}
	}

	return steps, handoffs
}

func TestTeammateStepsAreLabelledAndKeptApartFromTheOrchestrator(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Look at it"),
		call("c1", "read_file", map[string]string{"path": "a.txt"}),
		report("r1", "Looked"),
		finish("Done"),
	)
	task := r.startTeam(t, "Look at the page")
	r.run(task.ID)

	events, _ := r.mem.EventsAfter(context.Background(), task.ID, 0)
	steps, handoffs := stepsBy(events, "Developer")
	if steps != 1 || !slices.Equal(handoffs, []string{"assigned", "reported"}) {
		t.Fatalf("steps carry the teammate's name and handoffs show both ways: %d %v", steps, handoffs)
	}

	messages, _ := r.mem.LoadTranscript(context.Background(), task.ID)
	for _, message := range messages {
		for _, block := range message.Blocks {
			if block.Kind == domain.BlockToolUse && hasAny([]string{block.Name}, "read_file", "report") {
				t.Fatalf("the teammate's calls must not reach the orchestrator's transcript: %+v", block)
			}
		}
	}
}

func TestToolsOfferedDifferByRole(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Work"), report("r1", "Done"), finish("Done"))
	task := r.startTeam(t, "Do it")
	r.run(task.ID)

	leadTools, mateTools := toolNames(r.model.requests[0]), toolNames(r.model.requests[1])
	if !hasAny(leadTools, "delegate") || !hasAny(leadTools, "finish") || hasAny(leadTools, "report") {
		t.Fatalf("the orchestrator delegates and finishes: %v", leadTools)
	}

	if !slices.Contains(mateTools, "report") || hasAny(mateTools, "delegate", "finish", "ask_user", "update_plan") {
		t.Fatalf("a teammate only reports and cannot reach the person or delegate: %v", mateTools)
	}

	lead, mate := r.model.requests[0].System, r.model.requests[1].System
	if !strings.Contains(mate, "Developer") || !strings.Contains(lead, "developer") {
		t.Fatal("each side gets instructions for its role")
	}
}

func TestNewTaskDefaultsToSoloAndHonorsExplicitTeamMode(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()}, finish("Hello"))

	ctx := context.Background()

	task, err := r.service.Create(ctx, "u1", application.CreateInput{ProjectID: "proj", Prompt: "Just talk"})
	if err != nil || task.Team || !task.PlanFirst {
		t.Fatalf("a task without a mode uses Solo and keeps the planning default: %+v %v", task, err)
	}

	yes := true

	teamTask, err := r.service.Create(ctx, "u1", application.CreateInput{
		ProjectID: "proj", Prompt: "Use the team", Team: &yes,
	})
	if err != nil || !teamTask.Team {
		t.Fatalf("an explicit Orchestrator choice is stored on the task: %+v %v", teamTask, err)
	}
}
