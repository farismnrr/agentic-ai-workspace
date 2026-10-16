package domain

import (
	"slices"
	"strings"
	"testing"
)

func TestThePlannerCarriesItsPlaybook(t *testing.T) {
	skills := SkillsFor("planner")
	if len(skills) != 1 || skills[0].Name != "prd" || !strings.Contains(skills[0].Body, "ask_user") ||
		strings.HasPrefix(skills[0].Body, "---") {
		t.Fatalf("playbook = %+v", skills)
	}

	if len(SkillsFor("developer")) != 0 || len(SkillsFor("../x")) != 0 {
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

func TestTheDesignerHasThreeSkillsWithADescription(t *testing.T) {
	names := []string{}

	for _, skill := range SkillsFor("designer") {
		if skill.Description == "" || skill.Body == "" || strings.HasPrefix(skill.Body, "---") {
			t.Fatalf("skill %q is incomplete: %+v", skill.Name, skill)
		}

		names = append(names, skill.Name)
	}

	if !slices.Equal(names, []string{"baseline-ui", "design-brief", "frontend-design"}) {
		t.Fatalf("designer skills = %v", names)
	}
}
