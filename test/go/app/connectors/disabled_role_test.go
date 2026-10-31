package connectors_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestDisabledExternalRoleStopsCallsUntilRoleSwitch(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	access := apptest.Connect(t, env.App, cookie, client, "coding:read")
	workID := mcpBegin(t, &env, access, `{"summary":"Check role is still enabled","role":"researcher"}`).Result.Structured.WorkID

	headers := apptest.JSONCookie(cookie)
	team := apptest.Do(t, env.App, "GET", "/api/agent-team", "", headers)
	var current struct {
		UseByDefault bool                     `json:"useByDefault"`
		Roles        []map[string]interface{} `json:"roles"`
	}
	if err := json.Unmarshal(team.Body, &current); err != nil {
		t.Fatalf("load team: %v %s", err, team.Body)
	}
	changed := false
	for _, role := range current.Roles {
		if role["key"] == "researcher" {
			role["enabled"] = false
			changed = true
		}
	}
	if !changed {
		t.Fatal("researcher role not found")
	}
	payload, _ := json.Marshal(current)
	saved := apptest.Do(t, env.App, "PUT", "/api/agent-team", string(payload), headers)
	if saved.Status != 200 {
		t.Fatalf("disable role: %d %s", saved.Status, saved.Body)
	}
	denied := apptest.CallTool(t, env.App, access, "list_files",
		`{"work_id":"`+workID+`","path":"."}`)
	if !strings.Contains(string(denied.Body), `"isError":true`) ||
		!strings.Contains(string(denied.Body), "role is disabled or unavailable") {
		t.Fatalf("disabled role continued working: %s", denied.Body)
	}
	// set_role must work even though validating the old active role would now fail.
	recovery := apptest.CallTool(t, env.App, access, "set_role",
		`{"work_id":"`+workID+`","role":"developer"}`)
	if !strings.Contains(string(recovery.Body), `"changed":true`) {
		t.Fatalf("cannot switch away from disabled role: %s", recovery.Body)
	}
	allowed := apptest.CallTool(t, env.App, access, "list_files",
		`{"work_id":"`+workID+`","path":"."}`)
	if strings.Contains(string(allowed.Body), `"isError":true`) {
		t.Fatalf("enabled new role could not run: %s", allowed.Body)
	}
}
