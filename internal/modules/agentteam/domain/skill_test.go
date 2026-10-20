package domain

import (
	"slices"
	"strings"
	"testing"
)

func TestThePlannerCarriesItsPlaybook(t *testing.T) {
	skills := SkillsFor("planner")
	if len(skills) != 1 || skills[0].Name != "prd" || !skills[0].Always || !strings.Contains(skills[0].Body, "ask_user") ||
		strings.HasPrefix(skills[0].Body, "---") {
		t.Fatalf("playbook = %+v", skills)
	}

	if len(SkillsFor("designer-none")) != 0 || len(SkillsFor("../x")) != 0 {
		t.Fatal("a role without a playbook has none")
	}
}

func TestASavedTeamGetsTheCurrentAccessAndPlaybook(t *testing.T) {
	saved := Role{Key: "planner", Name: "Planner", Prompt: "old", Access: []string{AccessRead}, Enabled: true}
	old := Team{Roles: []Role{saved}}

	var planner Role

	for _, role := range old.WithBuiltIns().Roles {
		if role.Key == "planner" {
			planner = role
		}
	}

	if !slices.Contains(planner.Access, AccessAsk) || len(planner.Skills) == 0 || planner.Prompt != "old" {
		t.Fatalf("planner = %+v", planner)
	}
}

func allAlways(skills map[string]Skill, names ...string) bool {
	for _, name := range names {
		if !skills[name].Always {
			return false
		}
	}

	return true
}

func TestTheDesignerHasItsPlaybooksWithTheirCraft(t *testing.T) {
	t.Parallel()

	byName := map[string]Skill{}

	for _, skill := range SkillsFor("designer") {
		if skill.Description == "" || skill.Body == "" || strings.HasPrefix(skill.Body, "---") {
			t.Fatalf("skill %q is incomplete: %+v", skill.Name, skill)
		}

		byName[skill.Name] = skill
	}

	always := allAlways(byName, "designer-workflow", "nuxt-ui", "ui-templates")
	if len(byName) != 6 || !always || byName["design-brief"].Always {
		t.Fatalf("designer skills = %v", byName)
	}

	if !slices.Contains(byName["frontend-design"].Craft, "anti-ai-slop") {
		t.Fatalf("frontend-design loads the anti-slop rules: %v", byName["frontend-design"].Craft)
	}
}

func TestEveryLibraryItemOfARoleCanBeRead(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"designer", "reviewer"} {
		library := LibraryFor(key)
		if len(library) == 0 {
			t.Fatalf("%s has no library", key)
		}

		for _, ref := range library {
			if text, ok := ReferenceText(key, ref.Name); !ok || text == "" || ref.Summary == "" {
				t.Errorf("%s: %s cannot be read", key, ref.Name)
			}
		}
	}
}

func TestEverySkillAsksOnlyForCraftThatExists(t *testing.T) {
	t.Parallel()

	for _, skill := range append(SkillsFor("designer"), SkillsFor("reviewer")...) {
		for _, name := range skill.Craft {
			if _, ok := ReferenceText("designer", "craft/"+name); !ok {
				t.Errorf("%s asks for craft/%s, which does not exist", skill.Name, name)
			}
		}
	}
}

func TestTheLibraryIsLimitedToTheRolesGroups(t *testing.T) {
	t.Parallel()

	if text, ok := ReferenceText("designer", "style/minimal"); !ok || !strings.Contains(text, "### tokens.css") {
		t.Fatalf("the Designer reads a style family: %v", ok)
	}

	if text, ok := ReferenceText("designer", "template/saas-landing"); !ok || !strings.Contains(text, "### example.html") {
		t.Fatalf("and a template with its example: %v", ok)
	}

	denied := [][2]string{
		{"reviewer", "style/minimal"},
		{"planner", "craft/color"},
		{"designer", "../x/y"},
		{"designer", "style/../../x"},
		{"designer", "style/none"},
		{"designer", "craft"},
	}

	for _, denied := range denied {
		if _, ok := ReferenceText(denied[0], denied[1]); ok {
			t.Errorf("%s must not read %s", denied[0], denied[1])
		}
	}
}

func TestTheStackSecurityAndArchitecturePlaybooksAreAlwaysOn(t *testing.T) {
	t.Parallel()

	playbooks := map[string][]string{
		"developer": {"clean-code", "delivery-rules", "nuxt-app"}, "architect": {"architecture-review", "stack-defaults"},
		"security": {"security-review"}, "qa": {"qa-flows"}, "business-analyst": {"requirements"},
	}

	for key, want := range playbooks {
		names := []string{}

		for _, skill := range SkillsFor(key) {
			if !skill.Always {
				t.Errorf("%s: %s must always be on", key, skill.Name)
			}

			names = append(names, skill.Name)
		}

		slices.Sort(names)

		if !slices.Equal(names, want) {
			t.Errorf("%s playbooks = %v, want %v", key, names, want)
		}
	}
}

func TestTheDesignerLooksAndTheReviewerChecksTheLook(t *testing.T) {
	t.Parallel()

	for _, role := range Presets().Members() {
		if role.Key == "designer" && !role.LooksAtScreens {
			t.Error("the Designer must look at the running app before it reports")
		}

		if role.Key == "reviewer" && (!role.ChecksLook || slices.Contains(role.Access, AccessWrite)) {
			t.Errorf("the Reviewer checks the look and cannot write: %+v", role)
		}
	}
}

func TestSecurityAndArchitectCheckButCannotWrite(t *testing.T) {
	t.Parallel()

	for _, role := range Presets().Members() {
		switch role.Key {
		case "security":
			if !role.ChecksSecurity || slices.Contains(role.Access, AccessWrite) {
				t.Errorf("the Security role checks and cannot write: %+v", role)
			}
		case "architect":
			if !role.ChecksArchitecture || slices.Contains(role.Access, AccessWrite) {
				t.Errorf("the Architect verifies and cannot write: %+v", role)
			}
		}
	}
}
