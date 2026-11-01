package execution_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// Treat only known historical shipped defaults as replaceable. A person's
// customized instructions and model preferences remain untouched.
func TestMergedRoleDefaultsUpgradeWithoutOverwritingCustomPrompts(t *testing.T) {
	a := apptest.NewAgentRig(t)
	headers := apptest.JSONCookie(a.Cookie)
	current := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", headers)
	var team struct {
		Roles []map[string]any `json:"roles"`
	}
	if err := json.Unmarshal(current.Body, &team); err != nil {
		t.Fatalf("decode team: %v", err)
	}
	const oldPlanner = "You handle business analysis and planning. Clarify requirements, users, risks and acceptance " +
		"criteria, then write a concise PRD and steps. Do not edit code, run tests or deploy."
	const oldSecurity = "You are an application security reviewer. Check the changed security boundary and its relevant " +
		"critical scenarios. Broaden dependency, static and failure-path checks when impact or saved task mode " +
		"requires it. Report exact checks, findings and uncertainty in plain words. Do not run unrelated full suites " +
		"for an ordinary change. Start your report with exactly Verdict: PASS or Verdict: FIX FIRST. Do not change " +
		"the code yourself."
	for _, role := range team.Roles {
		switch role["key"] {
		case "planner":
			role["prompt"] = oldPlanner
			role["purpose"] = "Gathers requirements and acceptance criteria, then writes a short PRD and plan."
			role["tier"] = "thinking"
		case "security":
			role["prompt"] = "My custom Security checklist."
			role["purpose"] = "Checks sensitive changes for vulnerabilities, risky code and missing protections."
			role["tier"] = "fast"
		}
	}
	save := func() []map[string]any {
		t.Helper()
		body, _ := json.Marshal(team)
		res := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", string(body), headers)
		if res.Status != 200 {
			t.Fatalf("save legacy team: %d %s", res.Status, res.Body)
		}
		var returned struct {
			Roles []map[string]any `json:"roles"`
		}
		if err := json.Unmarshal(res.Body, &returned); err != nil {
			t.Fatal(err)
		}
		return returned.Roles
	}
	verify := func(roles []map[string]any, expectSecurityCustom bool) {
		t.Helper()
		for _, role := range roles {
			switch role["key"] {
			case "planner":
				if strings.Contains(role["prompt"].(string), "business analysis and planning") ||
					!strings.Contains(role["prompt"].(string), "early architecture design") ||
					role["tier"] != "thinking" {
					t.Fatalf("legacy Planner not safely upgraded: %+v", role)
				}
			case "security":
				if expectSecurityCustom {
					if role["prompt"] != "My custom Security checklist." || role["tier"] != "fast" {
						t.Fatalf("custom Security instruction/model was overwritten: %+v", role)
					}
				} else if !strings.Contains(role["prompt"].(string), "combined Security + Architecture") {
					t.Fatalf("old Security default not upgraded: %+v", role)
				}
			case "architect":
				t.Fatal("retired Architect reappeared")
			}
		}
	}
	roles := save()
	verify(roles, true)
	// Simulate an unchanged saved Security default from the prior release.
	team.Roles = roles
	for _, role := range team.Roles {
		if role["key"] == "security" {
			role["prompt"] = oldSecurity
		}
	}
	upgraded := save()
	verify(upgraded, false)
	team.Roles = upgraded
	verify(save(), false) // repeated normalization is idempotent
}
