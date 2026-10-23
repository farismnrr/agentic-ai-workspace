package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Scope keeps a teammate's blocks under their own key, apart from the orchestrator's.
func (m *memory) Scope(scope string) application.TranscriptStore {
	return scoped{mem: m, scope: scope}
}

type scoped struct {
	mem   *memory
	scope string
}

func (s scoped) AppendBlock(ctx context.Context, taskID string, role domain.Role, block domain.Block) error {
	return s.mem.AppendBlock(ctx, taskID+"#"+s.scope, role, block)
}

func (s scoped) LoadTranscript(ctx context.Context, taskID string) ([]domain.Message, error) {
	return s.mem.LoadTranscript(ctx, taskID+"#"+s.scope)
}

type fakeTeam struct {
	roster application.Roster
}

func (f *fakeTeam) Roster(context.Context, string) (application.Roster, error) { return f.roster, nil }

type fakeDelegations struct {
	rows map[string]application.Delegation
	mu   sync.Mutex
}

func newDelegations() *fakeDelegations {
	return &fakeDelegations{rows: map[string]application.Delegation{}}
}

func (f *fakeDelegations) StartDelegation(
	_ context.Context, taskID string, d application.Delegation, _ time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rows[taskID+"/"+d.ID] = d

	return nil
}

func (f *fakeDelegations) FindDelegation(_ context.Context, taskID, id string) (application.Delegation, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, ok := f.rows[taskID+"/"+id]

	return d, ok, nil
}

func (f *fakeDelegations) SetDelegationStatus(_ context.Context, taskID, id, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	d := f.rows[taskID+"/"+id]
	d.Status = status
	f.rows[taskID+"/"+id] = d

	return nil
}

func (f *fakeDelegations) EndDelegation(_ context.Context, taskID, id, status, report string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	d := f.rows[taskID+"/"+id]
	d.Status, d.Report = status, report
	f.rows[taskID+"/"+id] = d

	return nil
}

func (f *fakeDelegations) status(taskID string) string {
	d, _, _ := f.FindDelegation(context.Background(), taskID, "d1")

	return d.Status
}

// teamRig is a rig whose task leads a team.
type teamRig struct {
	*rig
	delegations *fakeDelegations
}

func developer() application.AgentRole {
	return application.AgentRole{
		Key: "developer", Name: "Developer", Purpose: "Writes the code.", Prompt: "Be careful.",
		ProviderID: "prov", Model: "dev-model", Tools: []string{"read_file", "edit_file", "run_task"},
	}
}

func researcher() application.AgentRole {
	return application.AgentRole{
		Key: "researcher", Name: "Researcher", Purpose: "Reads the project.", Tools: []string{"read_file"},
	}
}

func newTeamRig(limits func(*application.Limits), members []application.AgentRole, steps ...step) *teamRig {
	delegations := newDelegations()
	rig := newRigWith(nil, func(deps *application.Deps, lim *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true, Prompt: "Lead well."}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: members}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations

		if limits != nil {
			limits(lim)
		}
	}, steps...)

	return &teamRig{rig: rig, delegations: delegations}
}

// startTeam starts a team task that builds without waiting for a yes after the PRD.
func (r *teamRig) startTeam(t *testing.T, prompt string) domain.Task {
	t.Helper()

	no := false

	return r.startWith(t, prompt, &no)
}

// startPlanned starts a team task in plan mode, the default: nobody builds before the person says yes to the PRD.
func (r *teamRig) startPlanned(t *testing.T, prompt string) domain.Task {
	t.Helper()

	return r.startWith(t, prompt, nil)
}

func (r *teamRig) startWith(t *testing.T, prompt string, planFirst *bool) domain.Task {
	t.Helper()

	yes := true

	task, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: prompt, Team: &yes, PlanFirst: planFirst,
	})
	if err != nil {
		t.Fatal(err)
	}

	return task
}

func delegate(id, role, brief string) step {
	return call(id, "delegate", map[string]string{"role": role, "brief": brief})
}

func report(id, summary string) step {
	return call(id, "report", map[string]string{"summary": summary})
}
