package domain_test

import (
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
)

func TestACustomRoleThatCanChangeThingsCannotReadTheWeb(t *testing.T) {
	t.Parallel()

	team, err := domain.Team{Roles: []domain.Role{
		{Key: "orchestrator", Name: "Orchestrator", Lead: true},
		{Name: "Scout", Enabled: true, Access: []string{"web", "images"}},
		{Name: "Risky", Enabled: true, Access: []string{"web", "exec", "images"}},
	}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}

	var scout, risky domain.Role

	for _, role := range team.Roles {
		switch role.Name {
		case "Scout":
			scout = role
		case "Risky":
			risky = role
		}
	}

	readsWeb := slices.Contains(scout.Access, "web") && !slices.Contains(risky.Access, "web")
	if !readsWeb || !slices.Contains(risky.Access, "images") {
		t.Fatalf("web needs a role that cannot change anything: scout %v, risky %v", scout.Access, risky.Access)
	}
}
