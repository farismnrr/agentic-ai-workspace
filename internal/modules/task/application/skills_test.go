package application_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func designer() application.AgentRole {
	return application.AgentRole{
		Key: "designer", Name: "Designer", Purpose: "Designs the interface.", Tools: []string{"read_file"},
		Skills: []application.Skill{
			{Name: "design-brief", Description: "Write the DESIGN.md.", Body: "# Brief steps"},
			{Name: "frontend-design", Description: "Build the UI.", Body: "# Build steps"},
		},
	}
}

func TestATeammateWithSeveralSkillsPicksOneWithUseSkill(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{designer()},
		delegate("d1", "designer", "Design the landing page"),
		call("u1", "use_skill", map[string]string{"name": "frontend-design"}),
		call("u2", "use_skill", map[string]string{"name": "nope"}),
		report("r1", "Designed"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a landing page")
	r.run(task.ID)

	names := []string{}
	for _, tool := range r.model.requests[1].Tools {
		names = append(names, tool.Name)
	}

	system := r.model.requests[1].System
	if !slices.Contains(names, "use_skill") || !strings.Contains(system, "- design-brief: Write the DESIGN.md.") {
		t.Fatalf("the Designer is offered use_skill and sees what each skill is for: %v", names)
	}

	if loaded := resultFor(r.model.requests[2], "u1"); loaded.Text != "# Build steps" || loaded.IsError {
		t.Fatalf("use_skill returns the playbook: %+v", loaded)
	}

	got := resultFor(r.model.requests[3], "u2")
	if !got.IsError || !strings.Contains(got.Text, "design-brief, frontend-design") {
		t.Fatalf("an unknown skill lists the real ones: %+v", got)
	}

	if strings.Contains(system, "# Build steps") {
		t.Fatal("the body of a skill is loaded on request, not put in the prompt")
	}
}
