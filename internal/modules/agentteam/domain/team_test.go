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
	if !ok || lead.Key != "orchestrator" || len(keys) != 9 || !slices.Contains(keys, "planner") {
		t.Fatalf("expected the 9 SDLC roles led by the orchestrator: %v", keys)
	}

	if len(team.Members()) != 8 {
		t.Fatalf("the lead hands work to the other eight: %d", len(team.Members()))
	}

	if _, err := team.Normalize(); err != nil {
		t.Fatalf("the built-in team must be valid: %v", err)
	}
}

func TestPresetAccessFollowsTheRole(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"planner": {"read"}, "researcher": {"read"}, "business-analyst": {"read"}, "architect": {"read"},
		"reviewer": {"read"},
		"qa":       {"read", "exec"}, "developer": {"read", "write", "exec"}, "devops": {"read", "write", "exec"},
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
		{Name: "Boss", Lead: true},
		{Name: "Data Scientist", Enabled: true, Access: []string{"exec"}},
		{Name: "Data Scientist", Enabled: true},
	}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	got := keysOf(team)[1:]
	if !slices.Equal(got, []string{"data-scientist", "data-scientist-2"}) || !team.Roles[0].Enabled {
		t.Fatalf("keys come from names and stay unique, the lead is always on: %v", got)
	}

	if !slices.Equal(team.Roles[1].Access, []string{"read", "exec"}) {
		t.Fatalf("read is always included: %v", team.Roles[1].Access)
	}
}

func TestNormalizeRefusesBadTeams(t *testing.T) {
	t.Parallel()

	lead := domain.Role{Key: "lead", Name: "Lead", Lead: true}
	cases := map[string]domain.Team{
		"no roles":       {},
		"no lead":        {Roles: []domain.Role{{Name: "A"}}},
		"two leads":      {Roles: []domain.Role{lead, {Key: "b", Name: "B", Lead: true}}},
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

func TestModelNeedsBothProviderAndName(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{{Name: "Lead", Lead: true, Model: "only-model"}}}.Normalize()
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
	want := [][]string{{"read"}, {"read", "write", "exec"}, {"read", "write"}}

	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("role %d: access %v, want %v", i, got[i], want[i])
		}
	}
}
