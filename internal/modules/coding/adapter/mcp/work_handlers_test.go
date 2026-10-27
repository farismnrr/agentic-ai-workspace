package mcp

import (
	"context"
	"errors"
	"testing"

	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestAllWorkHandlersUseVerifiedIdentityAndReconnectState(t *testing.T) {
	for _, client := range []string{"ChatGPT", "Gemini"} {
		t.Run(client, func(t *testing.T) { exerciseWorkHandlers(t, client) })
	}
}

//nolint:gocognit,gocyclo,cyclop // one ordered scenario exercises the full external work lifecycle.
func exerciseWorkHandlers(t *testing.T, client string) {
	works := &fakeWorkSessions{}
	tools := &tools{workspaces: fakeWorkspaces{}, works: works}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{User: "u1", Client: client})

	begin, _, err := tools.beginWork(ctx, beginWorkInput{Summary: "Build feature", Role: "developer"})
	if err != nil || begin.WorkID != "work-1" || begin.TaskID != "task-1" {
		t.Fatalf("begin = %+v, %v", begin, err)
	}
	target := WorkTarget{WorkID: begin.WorkID}
	role, _, callErr := tools.setRole(ctx, roleWorkInput{WorkTarget: target, Role: "Developer"})
	if callErr != nil || !role.Changed {
		t.Fatalf("set role = %+v, %v", role, callErr)
	}
	plan, _, err := tools.updatePlan(ctx, workPlanInput{
		WorkTarget: target, BaseRevision: 0,
		RequestID: "request-123", Steps: []PlanWorkStep{{Title: "Implement", State: "pending"}},
	})
	if err != nil || plan.PlanRevision != 1 || len(plan.Steps) != 1 {
		t.Fatalf("update plan = %+v, %v", plan, err)
	}
	research, _, callErr := tools.saveResearch(ctx, researchInput{WorkTarget: target, Markdown: "Finding"})
	if callErr != nil || !research.Saved {
		t.Fatalf("save research = %+v, %v", research, callErr)
	}
	requirements, _, callErr := tools.saveRequirements(ctx, markdownInput{WorkTarget: target, Markdown: "Requirement"})
	if callErr != nil || !requirements.Saved {
		t.Fatalf("save requirements = %+v, %v", requirements, callErr)
	}
	progress, _, callErr := tools.reportProgress(ctx, progressInput{
		WorkTarget: target, Message: "Done", Claims: []string{"tested"},
	})
	if callErr != nil || progress.State != "open" {
		t.Fatalf("report progress = %+v, %v", progress, callErr)
	}
	complete, _, callErr := tools.completeWork(ctx, completeInput{
		WorkTarget: target, Outcome: "completed", Summary: "Finished",
	})
	if callErr != nil || complete.TaskStatus != "done" {
		t.Fatalf("complete = %+v, %v", complete, callErr)
	}
	first, _, err := tools.getWorkState(ctx, workIDInput{WorkTarget: target})
	if err != nil || first.WorkID != begin.WorkID || first.Warnings == nil {
		t.Fatalf("state = %+v, %v", first, err)
	}
	secondCtx := context.WithValue(t.Context(), identityKey{}, identity{User: "u1", Client: client})
	second, _, err := tools.getWorkState(secondCtx, workIDInput{WorkTarget: target})
	if err != nil || second.WorkID != first.WorkID || second.State != first.State {
		t.Fatalf("reconnected state = %+v, %v", second, err)
	}
	if works.lastUser != "u1" || works.lastClient != client || works.lastProject != "project-1" {
		t.Fatalf("identity came from tool input: %q %q %q", works.lastUser, works.lastClient, works.lastProject)
	}
}

func TestWorkIDValidationHidesOwnerClientAndProjectMismatches(t *testing.T) {
	for _, tc := range []struct {
		who     identity
		project string
	}{
		{who: identity{User: "other", Client: "ChatGPT"}, project: "project-1"},
		{who: identity{User: "u1", Client: "Other"}, project: "project-1"},
		{who: identity{User: "u1", Client: "ChatGPT"}, project: "other-project"},
	} {
		ctx := context.WithValue(t.Context(), identityKey{}, tc.who)
		tools := &tools{workspaces: fakeWorkspaces{}, works: &fakeWorkSessions{}}
		_, _, err := tools.workContext(ctx, WorkTarget{WorkID: "work-1", Project: tc.project})
		if !errors.Is(err, taskdomain.ErrUnknownWork) {
			t.Fatalf("identity %+v/project %q error = %v", tc.who, tc.project, err)
		}
	}
}

type fakeWorkspaces struct{}

func (fakeWorkspaces) For(_ context.Context, _, _ string, project string) (Services, error) {
	if project == "" {
		project = "project-1"
	}
	return Services{ProjectID: project}, nil
}

func (fakeWorkspaces) Projects(context.Context, string, string) ([]ProjectInfo, error) {
	return nil, nil
}

//nolint:govet // fieldalignment: keep fake service fields grouped by recorded concern.
type fakeWorkSessions struct {
	WorkSessions
	lastUser      string
	lastClient    string
	lastProject   string
	research      string
	requirements  string
	state         WorkState
	roles         []string
	validations   int
	approvalCalls int
}

func (f *fakeWorkSessions) Validate(_ context.Context, user, client, project, workID string) error {
	f.validations++
	f.lastUser, f.lastClient, f.lastProject = user, client, project
	if user != "u1" || (client != "ChatGPT" && client != "Gemini") || project != "project-1" || workID != "work-1" {
		return taskdomain.ErrUnknownWork
	}
	return nil
}

func (f *fakeWorkSessions) Begin(_ context.Context, _, _, _ string, _ BeginWorkInput) (BeginWorkResult, error) {
	f.state = WorkState{WorkID: "work-1", State: "open", Warnings: []string{}}
	return BeginWorkResult{TaskID: "task-1", WorkID: "work-1", State: "open", Warnings: []string{}}, nil
}

func (f *fakeWorkSessions) AvailableRoles(context.Context, string) ([]WorkRole, error) {
	return []WorkRole{{Key: "developer", Name: "Developer"}, {Key: "researcher", Name: "Researcher"}}, nil
}

func (f *fakeWorkSessions) ApprovePlan(_ context.Context, user, client, project, workID string,
	revision int64, _ string,
) (PlanApprovalResult, error) {
	f.approvalCalls++
	f.lastUser, f.lastClient, f.lastProject = user, client, project
	return PlanApprovalResult{
		Decision: "approved", HandoffID: "handoff-1", HandoffState: "done",
		InternalTaskID: "internal-1", Revision: revision,
	}, nil
}

func (f *fakeWorkSessions) SetRole(_ context.Context, _, _, _, workID, role, _ string) (RoleWorkResult, error) {
	f.state.Role = role
	f.roles = append(f.roles, role)
	return RoleWorkResult{
		WorkID: workID, Revision: f.state.Revision, State: f.state.State,
		Role: WorkRole{Key: "developer", Name: role}, Changed: true, Warnings: []string{},
	}, nil
}

func (f *fakeWorkSessions) UpdatePlan(
	_ context.Context, _, _, _, workID string, in PlanWorkInput,
) (PlanWorkResult, error) {
	f.state.Plan = in.Steps
	if f.state.PlanState == "ready_for_approval" {
		f.state.PlanState = "draft"
		if f.state.Review != nil {
			f.state.Review.Status = "superseded"
		}
	}
	f.state.Revision++
	return PlanWorkResult{
		WorkID: workID, Revision: f.state.Revision, State: f.state.State,
		PlanRevision: f.state.Revision, Steps: in.Steps, Warnings: []string{},
	}, nil
}

func (f *fakeWorkSessions) SaveResearch(
	_ context.Context, _, _, _, workID string, input ResearchWorkInput,
) (WorkMutation, error) {
	f.research = input.Markdown
	return mutation(workID, f.state), nil
}

func (f *fakeWorkSessions) SaveRequirements(_ context.Context, _, _, _, workID, markdown string) (WorkMutation, error) {
	f.requirements = markdown
	return mutation(workID, f.state), nil
}

func (f *fakeWorkSessions) Progress(
	_ context.Context, _, _, _, workID string, _ ProgressWorkInput,
) (WorkMutation, error) {
	return mutation(workID, f.state), nil
}

func (f *fakeWorkSessions) State(_ context.Context, _, _, _, workID string) (WorkState, error) {
	return f.state, nil
}

func mutation(workID string, state WorkState) WorkMutation {
	return WorkMutation{WorkID: workID, Revision: state.Revision, State: state.State, Warnings: []string{}}
}
