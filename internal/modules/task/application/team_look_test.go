package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func lookingDesigner() application.AgentRole {
	return application.AgentRole{
		Key: "designer", Name: "Designer", Purpose: "Designs.", LooksAtScreens: true,
		Tools: []string{"read_file", "edit_file", "screenshot"},
	}
}

func screenReviewer() application.AgentRole {
	return application.AgentRole{
		Key: "reviewer", Name: "Reviewer", Purpose: "Looks.", Tools: []string{"read_file", "screenshot"}, ChecksLook: true,
	}
}

// newShotRig is newTeamRig with a screenshot store, so the screenshot tool leaves a picture in the thread.
func newShotRig(members []application.AgentRole, steps ...step) *teamRig {
	delegations := newDelegations()
	shots := &memoryShots{saved: map[string][]byte{}}
	rig := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true, Prompt: "Lead well."}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: members}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations
		deps.Shots = shots
	}, steps...)
	rig.tools.risks["edit_file"] = domain.RiskLow
	rig.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"css/style.css"}}
	rig.tools.results["screenshot"] = application.ToolResult{Output: "Screenshot of /", Image: []byte("PNG")}
	rig.gate.skip = true

	return &teamRig{rig: rig, delegations: delegations}
}

func TestADesignerWhoChangedScreensMustLookAtThemBeforeReporting(t *testing.T) {
	t.Parallel()

	r := newShotRig([]application.AgentRole{lookingDesigner()},
		delegate("d1", "designer", "Style the pages"),
		call("e1", "edit_file", map[string]string{"path": "css/style.css"}),
		report("r1", "Styled it"),
		call("s1", "screenshot", map[string]any{}),
		report("r2", "Styled it and looked at it"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make it look right")
	r.run(task.ID)

	refused := resultFor(r.model.requests[3], "r1")
	if !refused.IsError || !strings.Contains(refused.Text, "screenshot") {
		t.Fatalf("the first report is sent back: %+v", refused)
	}

	if r.lastEvent(task.ID, domain.EventShot).Agent != "Designer" {
		t.Fatal("the Designer looked at the app and the picture is in the thread")
	}
}

func TestTheReviewerLooksAtTheScreensBeforeTheTaskIsDone(t *testing.T) {
	t.Parallel()

	r := newShotRig([]application.AgentRole{screenReviewer(), buildingDeveloper()},
		delegate("d1", "developer", "Style it"),
		call("e1", "edit_file", map[string]string{"path": "css/style.css"}),
		report("r1", "Styled"),
		finish("Too early"),
		delegate("d2", "reviewer", "Design review"),
		report("r2", "Looks right"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a page")
	r.run(task.ID)

	held := resultFor(r.model.requests[4], "fin")
	if !held.IsError || !strings.Contains(held.Text, "Reviewer") || !strings.Contains(held.Text, "design review") {
		t.Fatalf("the first finish waits for the Reviewer: %+v", held)
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("after the Reviewer reported the task finishes: %q", got)
	}
}
