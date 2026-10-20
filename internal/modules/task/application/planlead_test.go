package application_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// scriptedProgram is a vendor's program whose turn is a fixed set of calls to the server's tools.
type scriptedProgram struct {
	say      string
	requests []application.CLIRequest
	answers  []string
	calls    []scriptedCall
	failed   []bool
}

type scriptedCall struct {
	input any
	name  string
}

func (*scriptedProgram) Ready(context.Context, string, string) (bool, error) { return true, nil }

func (p *scriptedProgram) Run(
	ctx context.Context, req application.CLIRequest, emit func(application.CLIEvent),
) (application.CLIResult, error) {
	p.requests = append(p.requests, req)

	for _, c := range p.calls {
		raw, _ := json.Marshal(c.input)
		text, failed := req.Bridge.Call(ctx, c.name, raw)
		p.answers, p.failed = append(p.answers, text), append(p.failed, failed)
	}

	emit(application.CLIEvent{Kind: application.CLIText, Text: p.say})

	return application.CLIResult{Session: "s1"}, nil
}

type sessionBook map[string]string

func (b sessionBook) Session(_ context.Context, id string) (string, error) { return b[id], nil }

func (b sessionBook) SaveSession(_ context.Context, id, session string) error {
	b[id] = session

	return nil
}

func newPlanLeadRig(program *scriptedProgram, steps ...step) *teamRig {
	delegations := newDelegations()
	rig := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true, Prompt: "Lead well."}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: []application.AgentRole{developer()}}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations
		deps.CLI, deps.Sessions = program, sessionBook{}
	}, steps...)

	return &teamRig{rig: rig, delegations: delegations}
}

func (r *teamRig) startOnPlan(t *testing.T, prompt string) domain.Task {
	t.Helper()

	yes, no := true, false

	task, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: prompt, Team: &yes, PlanFirst: &no, ProviderID: "plan:claude", Model: "sonnet",
	})
	if err != nil {
		t.Fatal(err)
	}

	return task
}

func toolsOf(req application.CLIRequest) []string {
	names := []string{}

	for _, tool := range req.Bridge.Tools {
		names = append(names, tool.Name)
	}

	slices.Sort(names)

	return names
}

func TestALeadOnAPlanGetsTheServersToolsAndOnlyLooksAtFiles(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "Done."}
	r := newPlanLeadRig(program)
	r.tools.risks["edit_file"], r.tools.risks["run_command"] = domain.RiskMedium, domain.RiskHigh
	r.run(r.startOnPlan(t, "Fix the button").ID)

	req := program.requests[0]
	if got := toolsOf(req); !slices.Equal(got, []string{"ask_user", "delegate", "update_plan"}) {
		t.Fatalf("tools = %v", got)
	}

	if !req.Limited || slices.Contains(req.Tools, "edit_file") || !slices.Contains(req.Tools, "read_file") {
		t.Fatalf("the lead may only look: limited=%v tools=%v", req.Limited, req.Tools)
	}
}

func TestALeadOnAPlanHandsJobsToTeammatesThroughTheServersTools(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "All fixed.", calls: []scriptedCall{
		{name: "delegate", input: map[string]string{"role": "developer", "brief": "Fix the button"}},
	}}
	r := newPlanLeadRig(program,
		call("c1", "read_file", map[string]string{"path": "a.txt"}), report("r1", "The button is fixed"))
	task := r.startOnPlan(t, "Fix the button")
	r.run(task.ID)

	if program.failed[0] || !strings.Contains(program.answers[0], "The button is fixed") {
		t.Fatalf("the program got %q (failed %v)", program.answers[0], program.failed[0])
	}

	events, _ := r.mem.EventsAfter(context.Background(), task.ID, 0)
	if _, handoffs := stepsBy(events, "Developer"); !slices.Equal(handoffs, []string{"assigned", "reported"}) {
		t.Fatalf("handoffs = %v", handoffs)
	}

	for _, d := range r.delegations.rows {
		if d.Status != application.DelegationDone || !strings.HasPrefix(d.ID, "lead-") {
			t.Fatalf("delegation %+v", d)
		}
	}

	if got := r.task(t, task.ID).Status; len(r.delegations.rows) != 1 || got != domain.StatusDone {
		t.Fatalf("delegations %v, task %q", r.delegations.rows, got)
	}
}

func TestALeadOnAPlanThatAsksThePersonStopsAndGetsTheAnswerNextTurn(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "Asking.", calls: []scriptedCall{
		{name: "ask_user", input: map[string]any{"question": "Which page?", "options": []string{"Home", "Menu"}}},
	}}
	r := newPlanLeadRig(program)
	task := r.startOnPlan(t, "Change a page")
	r.run(task.ID)

	if got := r.task(t, task.ID).Status; got != domain.StatusNeedsInput {
		t.Fatalf("status = %q, want needs_input", got)
	}

	if !strings.Contains(program.answers[0], "End your turn") {
		t.Fatalf("the program was not told to stop: %q", program.answers[0])
	}

	events, _ := r.mem.EventsAfter(context.Background(), task.ID, 0)
	for _, event := range events {
		if event.Kind == domain.EventQuestion {
			event.Answered = "Menu"
			_ = r.mem.UpdateEvent(context.Background(), task.ID, event)
		}
	}

	program.calls = nil
	r.run(task.ID)

	if got := program.requests[1]; !strings.Contains(got.Prompt, "The person answered: Menu") || got.Session != "s1" {
		t.Fatalf("second turn: session %q prompt %q", got.Session, got.Prompt)
	}
}
