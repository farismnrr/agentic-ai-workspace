package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
)

func keysOf(team domain.Team) []string {
	keys := []string{}
	for _, role := range team.Roles {
		keys = append(keys, role.Key)
	}

	return keys
}

func TestPresetsAreTheSDLCTeamWithOneLead(t *testing.T) {
	t.Parallel()

	team := domain.Presets()
	lead, ok := team.Lead()

	keys := keysOf(team)
	if !ok || lead.Key != "orchestrator" || len(keys) != 13 || !slices.Contains(keys, "planner") {
		t.Fatalf("expected the 13 roles led by the orchestrator: %v", keys)
	}

	if len(team.Members()) != 12 {
		t.Fatalf("the lead hands work to the other twelve: %d", len(team.Members()))
	}

	if _, err := team.Normalize(); err != nil {
		t.Fatalf("the built-in team must be valid: %v", err)
	}
}

func TestPresetAccessFollowsTheRole(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"planner": {"read", "ask"}, "researcher": {"read", "web"}, "media": {"read", "write", "exec", "images"},
		"business-analyst": {"read", "ask"}, "architect": {"read"}, "reviewer": {"read"},
		"designer": {"read", "write", "exec", "ask"}, "documents": {"read", "write", "exec"},
		"qa": {"read", "exec"}, "security": {"read", "exec"},
		"developer": {"read", "write", "exec"}, "devops": {"read", "write", "exec"},
	}

	for _, role := range domain.Presets().Members() {
		if !slices.Equal(role.Access, want[role.Key]) {
			t.Errorf("%s: access %v, want %v", role.Key, role.Access, want[role.Key])
		}
	}
}

func TestNormalizeMintsUniqueKeysAndAlwaysKeepsRead(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{
		{Key: "orchestrator", Name: "Boss", Lead: true},
		{Name: "Data Scientist", Enabled: true, Access: []string{"exec"}},
		{Name: "Data Scientist", Enabled: true},
	}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	keys := keysOf(team)
	if !slices.Contains(keys, "data-scientist") || !slices.Contains(keys, "data-scientist-2") || !team.Roles[0].Enabled {
		t.Fatalf("keys come from names and stay unique, the lead is always on: %v", keys)
	}

	if mine := team.Roles[1]; !slices.Equal(mine.Access, []string{"read", "exec"}) {
		t.Fatalf("read is always included: %v", mine.Access)
	}
}

func TestNormalizeRefusesBadTeams(t *testing.T) {
	t.Parallel()

	lead := domain.Role{Key: "orchestrator", Name: "Orchestrator", Lead: true}
	cases := map[string]domain.Team{
		"no roles":       {},
		"duplicate key":  {Roles: []domain.Role{lead, {Key: "x", Name: "X"}, {Key: "x", Name: "Y"}}},
		"bad key":        {Roles: []domain.Role{lead, {Key: "No Spaces", Name: "X"}}},
		"empty name":     {Roles: []domain.Role{lead, {Name: " "}}},
		"long prompt":    {Roles: []domain.Role{lead, {Name: "X", Prompt: strings.Repeat("a", domain.MaxPrompt+1)}}},
		"unknown access": {Roles: []domain.Role{lead, {Name: "X", Access: []string{"root"}}}},
		"too many roles": {Roles: slices.Repeat([]domain.Role{{Name: "X"}}, domain.MaxRoles+1)},
	}

	for name, team := range cases {
		if _, err := team.Normalize(); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s: want ErrInvalid, got %v", name, err)
		}
	}
}

func TestOnlyTheOrchestratorLeads(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{{Name: "Pretender", Lead: true, Enabled: true}}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	lead, _ := team.Lead()
	if lead.Key != "orchestrator" {
		t.Fatalf("a role the person adds never leads: %+v", lead)
	}
}

func TestBuiltInRolesCannotBeRemoved(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{{Name: "Only mine", Enabled: true}}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	keys := keysOf(team)
	for _, built := range keysOf(domain.Presets()) {
		if !slices.Contains(keys, built) {
			t.Errorf("built-in role %s must stay on the team: %v", built, keys)
		}
	}

	if lead, ok := team.Lead(); !ok || lead.Key != "orchestrator" || !slices.Contains(keys, "only-mine") {
		t.Fatalf("the lead is restored and the person's own role stays: %v", keys)
	}
}

func TestOlderSavedTeamsGetNewBuiltInRoles(t *testing.T) {
	t.Parallel()

	old := domain.Team{Roles: []domain.Role{{Key: "orchestrator", Name: "Orchestrator", Lead: true, Enabled: true}}}
	if keys := keysOf(old.WithBuiltIns()); !slices.Contains(keys, "planner") || len(keys) != 13 {
		t.Fatalf("a team saved before the Planner existed gets it: %v", keys)
	}
}

func TestModelNeedsBothProviderAndName(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{{Key: "orchestrator", Name: "Lead", Model: "only-model"}}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	if team.Roles[0].Model != "" || team.Roles[0].ProviderID != "" {
		t.Fatalf("half a model choice means the default: %+v", team.Roles[0])
	}
}

func TestBuiltInRolesKeepTheirAccessWhateverIsSent(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{
		{Key: "orchestrator", Name: "Orchestrator", Lead: true},
		{Key: "researcher", Name: "Researcher", Enabled: true, Access: []string{"read", "write", "exec"}},
		{Key: "developer", Name: "Developer", Enabled: true, Access: []string{"read"}},
		{Name: "My own", Enabled: true, Access: []string{"write"}},
	}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	got := [][]string{team.Roles[1].Access, team.Roles[2].Access, team.Roles[3].Access}
	want := [][]string{{"read", "web"}, {"read", "write", "exec"}, {"read", "write"}}

	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("role %d: access %v, want %v", i, got[i], want[i])
		}
	}
}
