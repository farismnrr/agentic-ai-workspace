package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestModelTiersAndARolesTierAreSaved(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)

	body := `{"defaultProviderId":"p1","defaultModel":"sonnet","defaultEffort":"medium",` +
		`"thinking":{"providerId":"p1","model":"opus","effort":"high"},` +
		`"fast":{"providerId":"p1","model":"haiku","effort":"low"},` +
		`"personaName":"","talkStyle":"","language":"","about":"","rules":[]}`
	saved := do(t, env.app, "PUT", "/api/preferences", body, jsonCookie(cookie))
	if saved.Status != 200 {
		t.Fatalf("save failed: %d %s", saved.Status, saved.Body)
	}

	got := string(do(t, env.app, "GET", "/api/preferences", "", jsonCookie(cookie)).Body)
	wanted := []string{`"model":"opus"`, `"effort":"high"`, `"model":"haiku"`, `"effort":"low"`, `"defaultModel":"sonnet"`}

	for _, want := range wanted {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}

	team := do(t, env.app, "GET", "/api/agent-team", "", jsonCookie(cookie)).JSON(t)
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
	if res := do(t, env.app, "PUT", "/api/agent-team", string(encoded), jsonCookie(cookie)); res.Status != 200 {
		t.Fatalf("saving the team failed: %d %s", res.Status, res.Body)
	}

	again := string(do(t, env.app, "GET", "/api/agent-team", "", jsonCookie(cookie)).Body)
	if strings.Count(again, `"tier":"fast"`) != 1 || strings.Count(again, `"tier":"thinking"`) != 1 {
		t.Fatalf("the tiers of the roles must be kept: %s", again)
	}
}
