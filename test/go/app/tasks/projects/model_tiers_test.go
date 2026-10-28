package projects_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestModelTiersAndARolesTierAreSaved(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)

	body := `{"defaultProviderId":"p1","defaultModel":"sonnet","defaultEffort":"medium",` +
		`"thinking":{"providerId":"p1","model":"opus","effort":"high"},` +
		`"fast":{"providerId":"p1","model":"haiku","effort":"low"},` +
		`"personaName":"","talkStyle":"","language":"","about":"","rules":[]}`
	saved := apptest.Do(t, env.App, "PUT", "/api/preferences", body, apptest.JSONCookie(cookie))
	if saved.Status != 200 {
		t.Fatalf("save failed: %d %s", saved.Status, saved.Body)
	}

	got := string(apptest.Do(t, env.App, "GET", "/api/preferences", "", apptest.JSONCookie(cookie)).Body)
	wanted := []string{`"model":"opus"`, `"effort":"high"`, `"model":"haiku"`, `"effort":"low"`, `"defaultModel":"sonnet"`}

	for _, want := range wanted {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}

	team := apptest.Do(t, env.App, "GET", "/api/agent-team", "", apptest.JSONCookie(cookie)).JSON(t)
	roles, _ := team["roles"].([]any)

	for _, item := range roles {
		role, _ := item.(map[string]any)
		if role["key"] == "researcher" {
			role["tier"] = "fast"
		}

		if role["key"] == "architect" {
			role["tier"] = "thinking"
		}
	}

	encoded, _ := json.Marshal(team)
	if res := apptest.Do(t, env.App, "PUT", "/api/agent-team", string(encoded),
		apptest.JSONCookie(cookie)); res.Status != 200 {
		t.Fatalf("saving the team failed: %d %s", res.Status, res.Body)
	}

	again := string(apptest.Do(t, env.App, "GET", "/api/agent-team", "", apptest.JSONCookie(cookie)).Body)
	if strings.Count(again, `"tier":"fast"`) != 1 || strings.Count(again, `"tier":"thinking"`) != 1 {
		t.Fatalf("the tiers of the roles must be kept: %s", again)
	}
}
