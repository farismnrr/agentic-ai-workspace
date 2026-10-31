package execution_test

import (
	"encoding/json"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestLegacyBuiltInRolesMergeWithoutRestoringExecPrivileges(t *testing.T) {
	a := apptest.NewAgentRig(t)
	cookie := apptest.JSONCookie(a.Cookie)
	saved := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team",
		`{"roles":[
			{"key":"orchestrator","name":"Orchestrator","enabled":true},
			{"key":"designer","name":"Old Designer","enabled":true,"access":["write","exec"]},
			{"key":"media","name":"Old Media","enabled":true,"access":["write","exec","images"]},
			{"key":"documents","name":"Old Documents","enabled":true,"access":["write","exec"]},
			{"key":"reviewer","name":"Old Reviewer","enabled":true,"access":["read"]},
			{"key":"business-analyst","name":"Old BA","enabled":true,"access":["read","ask"]}
		]}`, cookie)
	if saved.Status != 200 {
		t.Fatalf("could not migrate saved team: %d %s", saved.Status, saved.Body)
	}
	for _, body := range [][]byte{
		saved.Body,
		apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", cookie).Body,
	} {
		var team struct {
			Roles []struct {
				Key    string   `json:"key"`
				Name   string   `json:"name"`
				Access []string `json:"access"`
			} `json:"roles"`
		}
		if err := json.Unmarshal(body, &team); err != nil {
			t.Fatalf("cannot decode roles: %v", err)
		}
		if len(team.Roles) != 8 {
			t.Fatalf("want eight merged roles, got %d: %s", len(team.Roles), body)
		}
		seen := map[string]bool{}
		for _, role := range team.Roles {
			if seen[role.Key] {
				t.Fatalf("duplicate merged role %s", role.Key)
			}
			seen[role.Key] = true
			switch role.Key {
			case "designer", "media", "documents", "reviewer", "business-analyst":
				t.Fatalf("retired role resurrected: %s", role.Key)
			case "developer":
				if role.Name != "Developer" || contains(role.Access, "exec") {
					t.Fatalf("developer inherited unsafe old Designer exec: %+v", role)
				}
			case "qa":
				if role.Name != "QA & Review" || contains(role.Access, "write") {
					t.Fatalf("QA inherited an unsafe old role: %+v", role)
				}
			case "planner":
				if role.Name != "Planner" || contains(role.Access, "write") {
					t.Fatalf("Planner inherited unsafe old BA access: %+v", role)
				}
			case "devops":
				if contains(role.Access, "write") {
					t.Fatalf("DevOps still owns source mutation: %+v", role)
				}
			}
		}
	}
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
