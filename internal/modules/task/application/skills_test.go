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
			{Name: "workflow", Description: "How you work.", Body: "# Workflow steps", Always: true},
			{Name: "design-brief", Description: "Write the DESIGN.md.", Body: "# Brief steps"},
			{Name: "frontend-design", Description: "Build the UI.", Body: "# Build steps", Craft: []string{"color"}},
		},
		Library: []application.Reference{{Name: "craft/color", Summary: "Palette rules."}},
		Lookup: func(name string) (string, bool) {
			return "# Color rules", name == "craft/color"
		},
	}
}

func designerRig(t *testing.T) (*teamRig, application.Request) {
	t.Helper()

	r := newTeamRig(nil, []application.AgentRole{designer()},
		delegate("d1", "designer", "Design the landing page"),
		call("u1", "use_skill", map[string]string{"name": "frontend-design"}),
		call("u2", "use_skill", map[string]string{"name": "nope"}),
		call("f1", "read_reference", map[string]string{"name": "craft/color"}),
		call("f2", "read_reference", map[string]string{"name": "craft/none"}),
		report("r1", "Designed"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a landing page")
	r.run(task.ID)

	return r, r.model.requests[1]
}

func TestATeammateIsOfferedItsToolsAndPromptLists(t *testing.T) {
	t.Parallel()

	_, request := designerRig(t)
	names := []string{}

	for _, tool := range request.Tools {
		names = append(names, tool.Name)
	}

	if !slices.Contains(names, "use_skill") || !slices.Contains(names, "read_reference") {
		t.Fatalf("the Designer is offered use_skill and read_reference: %v", names)
	}

	system := request.System
	listed := strings.Contains(system, "- design-brief: Write the DESIGN.md.")
	if !listed || !strings.Contains(system, "- craft/color: Palette rules.") {
		t.Fatalf("the prompt lists its skills and library: %q", system)
	}

	if strings.Contains(system, "# Build steps") || !strings.Contains(system, "# Workflow steps") {
		t.Fatal("an always-on skill is in the prompt; the others are loaded on request")
	}
}

func TestUseSkillAndReadReferenceAnswerByName(t *testing.T) {
	t.Parallel()

	r, _ := designerRig(t)

	loaded := resultFor(r.model.requests[2], "u1")
	withCraft := strings.HasPrefix(loaded.Text, "# Build steps") && strings.Contains(loaded.Text, "# Color rules")
	if !withCraft || loaded.IsError {
		t.Fatalf("use_skill returns the playbook with its craft: %+v", loaded)
	}

	if got := resultFor(r.model.requests[3], "u2"); !got.IsError || !strings.Contains(got.Text, "workflow, design-brief") {
		t.Fatalf("an unknown skill lists the real ones: %+v", got)
	}

	if resultFor(r.model.requests[4], "f1").Text != "# Color rules" || !resultFor(r.model.requests[5], "f2").IsError {
		t.Fatal("the library is read by full name")
	}
}
