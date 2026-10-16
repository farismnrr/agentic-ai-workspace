package domain

import (
	"slices"
	"strings"
	"testing"
)

func TestThePlannerCarriesItsPlaybook(t *testing.T) {
	skill := SkillFor("planner")
	if !strings.Contains(skill, "ask_user") || strings.HasPrefix(skill, "---") {
		t.Fatalf("playbook = %q", skill)
	}

	if SkillFor("developer") != "" || SkillFor("../x") != "" {
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

	if !slices.Contains(planner.Access, AccessAsk) || planner.Skill == "" || planner.Prompt != "old" {
		t.Fatalf("planner = %+v", planner)
	}
}
