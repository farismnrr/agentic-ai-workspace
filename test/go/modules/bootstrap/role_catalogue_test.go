package bootstrap_test

import (
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
	teamdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
)

// Regression contract: role ownership cannot be widened by saved prompts or skills.
func TestBuiltInRoleToolOwnershipMatrix(t *testing.T) {
	cases := map[string]struct {
		allowed []string
		denied  []string
	}{
		"orchestrator": {
			allowed: []string{"read_file", "search_code"},
			denied:  []string{"edit_file", "run_command", "git", "start_site", "browser_test"},
		},
		"planner": {
			allowed: []string{"read_file", "ask_user"},
			denied:  []string{"edit_file", "run_command", "run_build", "start_site"},
		},
		"researcher": {
			allowed: []string{"read_file", "git", "web_search"},
			denied:  []string{"edit_file", "run_command", "start_site", "browser_test", "gh"},
		},
		"developer": {
			allowed: []string{"write_file", "edit_file", "git", "gh", "run_artifact", "marimo_execute"},
			denied:  []string{"run_command", "run_build", "start_site", "verify", "browser_test"},
		},
		"qa": {
			allowed: []string{"verify", "browser_test", "read_browser_screenshot", "run_task", "gh"},
			denied:  []string{"run_command", "write_file", "run_build", "start_site", "git"},
		},
		"security": {
			allowed: []string{"read_file", "run_task"},
			denied:  []string{"run_command", "write_file", "run_build", "start_site", "git", "browser_test"},
		},
		"devops": {
			allowed: []string{"run_build", "start_site", "stop_site", "ci_logs", "git", "gh", "run_command"},
			denied:  []string{"edit_file", "write_file", "verify", "browser_test"},
		},
	}
	roles := teamdomain.Presets().Roles
	if len(roles) != len(cases) {
		t.Fatalf("built-in roles=%d, want %d", len(roles), len(cases))
	}
	for _, role := range roles {
		tc, ok := cases[role.Key]
		if !ok {
			t.Fatalf("unrecognized built-in role %s", role.Key)
		}
		t.Run(role.Key, func(t *testing.T) {
			tools := bootstrap.ToolsForRoleForTest(role)
			for _, name := range tc.allowed {
				if !slices.Contains(tools, name) {
					t.Errorf("%s missing permitted %s", role.Key, name)
				}
			}
			for _, name := range tc.denied {
				if slices.Contains(tools, name) {
					t.Errorf("%s improperly received %s", role.Key, name)
				}
			}
		})
	}
}
