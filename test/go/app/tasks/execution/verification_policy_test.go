package execution_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestSwiftOrdinaryCodeChangeNeedsNoAutomaticReviewRoles(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("write", "write_file", map[string]string{"path": "web/src/label.css", "content": ".label{}"}),
		apptest.FinishCall("Updated the label"),
	)
	id := a.Start(t, "Update the label")
	a.WaitApproving(t, id, "review")

	for _, event := range a.Thread(t, id) {
		if event.Kind == "handoff" {
			t.Fatalf("ordinary Swift work must not summon a review role: %+v", event)
		}
	}
}

func TestDeepReviewRequiresIndependentReviewer(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: update the label", "state": "active"},
			{"title": "Reviewer: independently review the change", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "developer", "brief": "Update the label"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "developer", "brief": "Update the label"}),
		apptest.ToolCall("w1", "write_file", map[string]string{
			"path": "web/src/label.css", "content": ".label { font-weight: 600; }",
		}),
		apptest.ToolCall("r1", "report", map[string]string{"summary": "Updated the label"}),
		apptest.FinishCall("Updated the label"),
		apptest.ToolCall("p2", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: update the label", "state": "done"},
			{"title": "Reviewer: independently review the change", "state": "pending"},
		}}),
		apptest.ToolCall("v0", "delegate", map[string]string{"role": "reviewer", "brief": "Independently review the label"}),
		apptest.ToolCall("v1", "delegate", map[string]string{"role": "reviewer", "brief": "Independently review the label"}),
		apptest.ToolCall("vr", "report", map[string]string{"summary": "Verdict: PASS\nThe label change is consistent."}),
		apptest.FinishCall("Updated and independently reviewed the label"),
	)
	configureReviewTeam(t, a, "developer", "reviewer")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Update the label","team":true,"mode":"deep-review"}`,
		apptest.JSONCookie(a.Cookie))
	if created.Status != 201 || created.JSON(t)["mode"] != "deep-review" {
		t.Fatalf("deep-review task was not saved: %d %s", created.Status, created.Body)
	}
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "review")
	assertNoScriptExhaustion(t, a, id)
	if !hasHandoff(a.Thread(t, id), "reported", "Reviewer") {
		t.Fatalf("deep-review must include an independent Reviewer report: %+v", a.Thread(t, id))
	}
}

func TestSecurityReviewRejectsMissingAndSupersededEvidence(t *testing.T) {
	for _, test := range missingEvidenceCases() {
		t.Run(test.name, func(t *testing.T) {
			runMissingEvidenceCase(t, test.latestFail, test.missingAssessment)
		})
	}
}

type missingEvidenceCase struct {
	name              string
	latestFail        bool
	missingAssessment bool
}

func missingEvidenceCases() []missingEvidenceCase {
	return []missingEvidenceCase{
		{name: "PASS_without_checks"},
		{name: "older_PASS_then_latest_FIX_FIRST", latestFail: true},
	}
}

func runMissingEvidenceCase(t *testing.T, latestFail, missingAssessment bool) {
	t.Helper()
	steps := missingEvidenceSteps(latestFail, missingAssessment)
	a := apptest.NewAgentRig(t, steps...)
	configureReviewTeam(t, a, "developer", "security")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Update token handling","team":true,"planFirst":false}`,
		apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)
	assertBlockedTask(t, a, id, "token.go")
	if latestFail && !hasHandoff(a.Thread(t, id), "reported", "Security") {
		t.Fatal("the latest FIX FIRST report must remain authoritative")
	}
}

func missingEvidenceSteps(latestFail, missingAssessment bool) []string {
	steps := []string{
		apptest.ToolCall("p", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: change token", "state": "pending"},
			{"title": "Security: review token", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "developer", "brief": "Change token"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "developer", "brief": "Change token"}),
		apptest.ToolCall("w", "write_file", map[string]string{
			"path": "internal/modules/auth/token.go", "content": "package auth",
		}),
		apptest.ToolCall("r", "report", map[string]string{"summary": "Updated"}),
		apptest.ToolCall("s1", "delegate", map[string]string{"role": "security", "brief": "Review token"}),
		apptest.ToolCall("s1r", "report", map[string]string{
			"summary": securityReport("Verdict: PASS\nNo checks were run", assessmentPaths(missingAssessment), nil),
		}),
	}
	if latestFail {
		steps = append(steps,
			apptest.ToolCall("s2", "delegate", map[string]string{"role": "security", "brief": "Review again"}),
			apptest.ToolCall("s2r", "report", map[string]string{
				"summary": securityReport("Verdict: FIX FIRST\nCritical issue remains.",
					assessmentPaths(false), nil),
			}),
		)
	}
	for range 8 {
		steps = append(steps, apptest.FinishCall("All checks passed"))
	}
	return steps
}

func configureReviewTeam(t *testing.T, a apptest.AgentRig, enabled ...string) {
	t.Helper()
	headers := apptest.JSONCookie(a.Cookie)
	team := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", headers).JSON(t)
	allowed := map[string]bool{"orchestrator": true}
	for _, role := range enabled {
		allowed[role] = true
	}
	for _, item := range team["roles"].([]any) {
		role := item.(map[string]any)
		key, _ := role["key"].(string)
		role["enabled"] = allowed[key]
		if key == "developer" && len(enabled) > 1 && enabled[1] == "security" {
			role["name"] = "Security"
		}
		if key == "security" && len(enabled) > 1 && enabled[1] == "security" {
			role["name"] = "Guard"
		}
	}
	encoded, _ := json.Marshal(team)
	if saved := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", string(encoded), headers); saved.Status != 200 {
		t.Fatalf("save review team: %d %s", saved.Status, saved.Body)
	}
}

func assertNoScriptExhaustion(t *testing.T, a apptest.AgentRig, id string) {
	t.Helper()
	for _, event := range a.Thread(t, id) {
		if event.Kind == "error" && strings.Contains(event.Text, "script ran out") {
			a.Gateway.Mu.Lock()
			requestCount := len(a.Gateway.Requests)
			a.Gateway.Mu.Unlock()
			t.Fatalf("script ran out after %d model requests: %s; events=%+v", requestCount, event.Text, a.Thread(t, id))
		}
	}
}

func assertBlockedTask(t *testing.T, a apptest.AgentRig, id, relevant string) {
	t.Helper()
	for _, event := range a.Thread(t, id) {
		if event.Kind == "error" && strings.Contains(event.Text, "blocked and unverified") &&
			strings.Contains(event.Text, relevant) {
			return
		}
	}
	t.Fatalf("task must finish blocked and unverified for %q; status=%s events=%+v",
		relevant, a.Status(t, id), a.Thread(t, id))
}

func hasHandoff(events []apptest.ThreadEvent, action, agent string) bool {
	for _, event := range events {
		if event.Kind == "handoff" && event.Action == action && event.Agent == agent {
			return true
		}
	}
	return false
}

func TestSensitiveChangeCannotFinishWithPlainModelReply(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("write", "write_file", map[string]string{
			"path": "server/api/profile.ts", "content": "export const profile = true",
		}),
		strings.Join([]string{
			`{"choices":[{"message":{"content":"The profile update is complete."}}],`,
			`"usage":{"prompt_tokens":30,"completion_tokens":10}}`,
		}, ""),
	)
	id := a.Start(t, "Update profile handling")
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)
	assertBlockedTask(t, a, id, "server/api/profile.ts")
}

func TestDeepReviewCannotFinishSensitiveChangeWithPlainReply(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("write", "write_file", map[string]string{
			"path": "internal/modules/auth/token.go", "content": "package auth\n",
		}),
		strings.Join([]string{
			`{"choices":[{"message":{"content":"The token change is complete."}}],`,
			`"usage":{"prompt_tokens":30,"completion_tokens":10}}`,
		}, ""),
	)
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Update token handling","mode":"deep-review","team":false}`,
		apptest.JSONCookie(a.Cookie))
	if created.Status != 201 || created.JSON(t)["mode"] != "deep-review" {
		t.Fatalf("deep-review task was not saved: %d %s", created.Status, created.Body)
	}
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)
	assertBlockedTask(t, a, id, "token.go")
}

func TestTrustedFollowupPromotionRollsBackWithMessageFailure(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	id := a.Start(t, "Explain the project")
	a.WaitFor(t, id, "done")
	headers := apptest.JSONCookie(a.Cookie)
	before := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers).Body
	query := "CREATE TRIGGER reject_followup BEFORE INSERT ON task_transcript " +
		"BEGIN SELECT RAISE(ABORT, 'test storage failure'); END"
	if _, err := a.Environment.DB.Exec(query); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{"text": "@deep-review\nContinue with review"})
	if err != nil {
		t.Fatal(err)
	}
	response := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages", string(body), headers)
	if response.Status != 500 {
		t.Fatalf("failed transcript persistence must fail the follow-up: %d %s", response.Status, response.Body)
	}
	after := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers).Body
	task := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "", headers).JSON(t)
	if string(before) != string(after) || task["mode"] != "swift" || task["status"] != "done" {
		t.Fatalf("failed follow-up left partial mode, message, or queue state: task=%v events=%s", task, after)
	}
}

func TestMultilineCommandRecordsWritesToIgnoredDirtyFiles(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("ignore", "write_file", map[string]string{"path": ".gitignore", "content": ".hidden-output\n"}),
		apptest.ToolCall("dirty", "write_file", map[string]string{"path": ".hidden-output", "content": "before"}),
		apptest.ToolCall("run", "run_command", map[string]string{
			"command": "printf 'generated' > .hidden-output\nprintf 'done'",
		}),
		apptest.FinishCall("The command completed"),
	)
	id := a.Start(t, "Run the project command")
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)

	if got := a.Read(".hidden-output"); got != "generated" {
		t.Fatalf("the command fixture did not create its file: %q", got)
	}
	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	if !strings.Contains(events, `"effectPaths":[".hidden-output"]`) ||
		!strings.Contains(events, `"effectsKnown":true`) || !strings.Contains(events, `unclassified impact`) {
		t.Fatalf("the ignored dirty file must be recorded and conservatively classified: %s", events)
	}
}

func TestCommandChangesToGitHooksAreVisibleToSecurityReview(t *testing.T) {
	t.Parallel()
	command := "mkdir -p .git/hooks && printf '#!/bin/sh\\nexit 0\\n' > .git/hooks/pre-commit"
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("run", "run_command", map[string]string{"command": command}),
		apptest.FinishCall("Updated the project hook"),
	)
	id := a.Start(t, "Update the project's Git hook")
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)
	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	if !strings.Contains(events, `"effectPaths":[`) || !strings.Contains(events, `".git/hooks/pre-commit"`) ||
		!strings.Contains(events, "blocked and unverified") {
		t.Fatalf("a real hook write must be captured and blocked for review: %s", events)
	}
}

func TestMutationAfterPassingCheckDoesNotRemainProvenInReceipt(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("check", "run_command", map[string]string{"command": "go test ./..."}),
		apptest.ToolCall("mutate", "run_command", map[string]string{
			"command": "printf 'changed after test\\n' > README.md",
		}),
		apptest.FinishCall("Updated the README after the check"),
	)
	a.PutFile(t, "go.mod", "module receipt\n\ngo 1.22\n")
	a.PutFile(t, "value.go", "package receipt\nfunc value() int { return 1 }\n")
	a.PutFile(t, "value_test.go", "package receipt\n\nimport \"testing\"\n\n"+
		"func TestValue(t *testing.T) { if value() != 1 { t.Fatal(\"bad\") } }\n")
	a.PutFile(t, "README.md", "baseline\n")
	id := a.Start(t, "Run a check, then update the README")
	a.WaitApproving(t, id, "review")
	assertNoScriptExhaustion(t, a, id)
	task := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "",
		apptest.JSONCookie(a.Cookie)).JSON(t)
	proof, _ := task["proof"].(map[string]any)
	if proof["proven"] != false {
		t.Fatalf("a check before a command mutation must not prove the final files: %v", proof)
	}
}

func TestReplacementGoTestExecutorsCannotProveSensitiveWork(t *testing.T) {
	for _, command := range []string{
		"go test -exec /bin/true ./...",
		"go test -exec=/bin/true ./...",
		`go test '-exec=/bin/true' ./...`,
		`go test '-exec' '/bin/true' ./...`,
		"go test --exec /bin/true ./...",
		"go test --exec=/bin/true ./...",
		`go test '--exec=/bin/true' ./...`,
		`go test '--exec' '/bin/true' ./...`,
		`go test --'exec'=/bin/true ./...`,
		`go test --'exec' '/bin/true' ./...`,
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			apptest.RequireDocker(t)
			a := apptest.NewAgentRig(t,
				apptest.ToolCall("write", "write_file", map[string]string{
					"path": "internal/modules/auth/token.go", "content": "package auth\n",
				}),
				apptest.ToolCall("fake-check", "run_command", map[string]string{"command": command}),
				apptest.FinishCall("Token handling is complete"),
			)
			a.PutFile(t, "go.mod", "module sample\n\ngo 1.22\n")
			a.PutFile(t, "token_test.go", "package sample\n\nimport \"testing\"\n\n"+
				"func TestPass(t *testing.T) {}\n")
			id := a.Start(t, "Update token handling")
			a.WaitApproving(t, id, "failed")
			assertNoScriptExhaustion(t, a, id)
			assertFakeExecutorDidNotProve(t, a, id, command)
		})
	}
}

func assertFakeExecutorDidNotProve(t *testing.T, a apptest.AgentRig, id, command string) {
	t.Helper()
	response := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "",
		apptest.JSONCookie(a.Cookie))
	proof, _ := response.JSON(t)["proof"].(map[string]any)
	// A blocked sensitive task may have no receipt; absence of proof is not success.
	if proof != nil && proof["proven"] != false {
		t.Fatalf("replacement executor must not prove the task: %v", proof)
	}
	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	if proof == nil && !strings.Contains(events, "blocked and unverified") {
		t.Fatalf("missing proof must be caused by the mandatory security gate: %s", events)
	}
	if !strings.Contains(events, `"command":"`+command+`"`) ||
		!strings.Contains(events, `"exitCode":0`) || strings.Contains(events, `"testExecuted":true`) {
		t.Fatalf("the successful wrapper must be visible but not count as executed tests: %s", events)
	}
}

func TestFailedCurrentGoCheckRemainsInReceiptAfterEarlierPass(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("pass", "run_command", map[string]string{"command": "go test -run TestPass ./..."}),
		apptest.ToolCall("edit", "write_file", map[string]string{"path": "README.md", "content": "Task edited\n"}),
		apptest.ToolCall("fail", "run_command", map[string]string{"command": "go test -run TestFail ./..."}),
		apptest.FinishCall("The task is complete"),
	)
	a.PutFile(t, "go.mod", "module receipt\n\ngo 1.22\n")
	a.PutFile(t, "value.go", "package receipt\nfunc value() int { return 1 }\n")
	a.PutFile(t, "value_test.go", "package receipt\n\nimport \"testing\"\n\n"+
		"func TestPass(t *testing.T) { if value() != 1 { t.Fatal(\"bad\") } }\n"+
		"func TestFail(t *testing.T) { t.Fatal(\"deliberate failure\") }\n")
	a.PutFile(t, "README.md", "baseline\n")
	id := a.Start(t, "Check the task, edit the README, then check again")
	a.WaitApproving(t, id, "review")
	assertNoScriptExhaustion(t, a, id)
	task := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "",
		apptest.JSONCookie(a.Cookie)).JSON(t)
	proof, _ := task["proof"].(map[string]any)
	checks, _ := proof["checks"].([]any)
	if proof["proven"] != false || !receiptShowsFailedCheck(checks) {
		t.Fatalf("the latest real failure must remain in the receipt: %v", proof)
	}
}

func receiptShowsFailedCheck(checks []any) bool {
	for _, item := range checks {
		check, _ := item.(map[string]any)
		if check["passed"] == false && strings.Contains(fmt.Sprint(check["text"]), "failed") {
			return true
		}
	}
	return false
}

func TestSensitiveChangeWithoutSecurityRoleEndsBlockedAndUnverified(t *testing.T) {
	t.Parallel()
	assertBlockedForImpact(t, "internal/modules/auth/token.go", "Update token handling",
		"Security review is mandatory", "token.go")
}

func TestSharedDependencyChangeRequiresAffectedCallerReview(t *testing.T) {
	t.Parallel()
	assertBlockedForImpact(t, "go.mod", "Update a dependency", "changes dependencies", "go.mod")
}

func TestRecordedPassingSecurityCheckAfterChangesSatisfiesEvidence(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: prepare checked package", "state": "active"},
			{"title": "Security: verify the package", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{
			"role": "developer", "brief": "Prepare a Go package for verification",
		}),
		apptest.ToolCall("d1", "delegate", map[string]string{
			"role": "developer", "brief": "Prepare a Go package for verification",
		}),
		apptest.ToolCall("t1", "write_file", map[string]string{
			"path": "sample_test.go",
			"content": "package sample\n\nimport \"testing\"\n\n" +
				"func TestValue(t *testing.T) { if 1+1 != 2 { t.Fatal(\"bad\") } }\n",
		}),
		apptest.ToolCall("d-report", "report", map[string]string{"summary": "Prepared the Go package"}),
		apptest.FinishCall("Prepared the Go package"),
		apptest.ToolCall("p2", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: prepare checked package", "state": "done"},
			{"title": "Security: verify the package", "state": "pending"},
		}}),
		apptest.ToolCall("s1", "delegate", map[string]string{
			"role": "security", "brief": "Run the Go package check and report its result",
		}),
		apptest.ToolCall("run", "run_command", map[string]string{"command": "go test ./..."}),
		apptest.ToolCall("s-report", "report", map[string]string{
			"summary": securityReport("Verdict: PASS\nChecks:\n- go test ./...",
				[]string{"sample_test.go"}, []string{"go test ./..."}),
		}),
		apptest.FinishCall("The package passed its required security check"),
		// A team can ask to reconcile its plan or save changes before review.
		apptest.ToolCall("final-plan", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: prepare checked package", "state": "done"},
			{"title": "Security: verify the package", "state": "done"},
		}}),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
		apptest.FinishCall("The package passed its required security check"),
	)
	// Keep go.mod as baseline: changing dependencies would also require an Architect review.
	a.PutFile(t, "go.mod", "module sample\n\ngo 1.22\n")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Verify the checked package","team":true,"mode":"swift"}`, apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "review")
	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	if !strings.Contains(events, `"command":"go test ./..."`) || !strings.Contains(events, `"effectsKnown":true`) {
		t.Fatalf("the full command and complete unchanged effect evidence must be recorded: %s", events)
	}
	if strings.Contains(events, `"kind":"error"`) {
		t.Fatalf("a current PASS with the recorded critical check must satisfy review: %s", events)
	}
}

func TestRenamedDeveloperCannotSatisfyStableSecurityReview(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("plan", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: change token handling", "state": "active"},
		}}),
		apptest.ToolCall("assign0", "delegate", map[string]string{
			"role": "developer", "brief": "Change token behavior and verify it with go test ./...",
		}),
		apptest.ToolCall("assign1", "delegate", map[string]string{
			"role": "developer", "brief": "Change token behavior and verify it with go test ./...",
		}),
		apptest.ToolCall("write", "write_file", map[string]string{
			"path":    "internal/modules/auth/token.go",
			"content": "package auth\n\nfunc Token() string { return \"valid\" }\n",
		}),
		apptest.ToolCall("check", "run_command", map[string]string{"command": "go test ./..."}),
		apptest.ToolCall("report", "report", map[string]string{
			"summary": securityReport("Verdict: PASS\nChecks:\n- go test ./...",
				[]string{"internal/modules/auth/token.go"}, []string{"go test ./..."}),
		}),
		strings.Join([]string{
			`{"choices":[{"message":{"content":"Token handling is complete."}}],`,
			`"usage":{"prompt_tokens":30,"completion_tokens":10}}`,
		}, ""),
		// Refused completion must exhaust the bounded review attempts.
		apptest.FinishCall("Token handling is complete"),
		apptest.FinishCall("Token handling is complete"),
		apptest.FinishCall("Token handling is complete"),
		apptest.FinishCall("Token handling is complete"),
	)
	a.PutFile(t, "go.mod", "module sample\n\ngo 1.22\n")
	a.PutFile(t, "internal/modules/auth/token_test.go", "package auth\n\nimport \"testing\"\n\n"+
		"func TestToken(t *testing.T) { if Token() != \"valid\" { t.Fatal(\"invalid token\") } }\n")
	configureReviewTeam(t, a, "developer", "security")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Update token handling","team":true,"planFirst":false}`,
		apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)
	assertBlockedTask(t, a, id, "token.go")
	assertDeveloperCheckReportPair(t, a, id)
}

func assertDeveloperCheckReportPair(t *testing.T, a apptest.AgentRig, id string) {
	t.Helper()
	var events []identityEvent
	response := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie))
	if err := json.Unmarshal(response.Body, &events); err != nil {
		t.Fatal(err)
	}
	reportJob, runJob, run := developerReportAndCheck(events)
	if reportJob != "assign1" || runJob != reportJob {
		t.Fatalf("the renamed Developer report and check must belong to assigned Developer job assign1: report=%q run=%q",
			reportJob, runJob)
	}
	if run.Command == "" || run.ExitCode != 0 || run.TimedOut || !run.EffectsKnown || len(run.EffectPaths) != 0 ||
		!run.TestExecuted {
		t.Fatalf("Developer evidence must be a genuine passing unchanged check: %+v", run)
	}
}

type identityEvent struct {
	Kind   string      `json:"kind"`
	Action string      `json:"action"`
	Agent  string      `json:"agent"`
	Job    string      `json:"job"`
	Run    identityRun `json:"run"`
}

type identityRun struct {
	Command      string   `json:"command"`
	EffectPaths  []string `json:"effectPaths"`
	ExitCode     int      `json:"exitCode"`
	TimedOut     bool     `json:"timedOut"`
	EffectsKnown bool     `json:"effectsKnown"`
	TestExecuted bool     `json:"testExecuted"`
}

func developerReportAndCheck(events []identityEvent) (string, string, identityRun) {
	var reportJob string
	var runJob string
	var runInfo identityRun
	for _, event := range events {
		if event.Kind == "handoff" && event.Action == "reported" && event.Agent == "Security" {
			reportJob = event.Job
		}
		if event.Kind == "action" && event.Agent == "Security" && event.Run.Command == "go test ./..." {
			runJob = event.Job
			runInfo = event.Run
		}
	}
	return reportJob, runJob, runInfo
}

type securityEvidenceCase struct {
	name              string
	commands          []string
	declared          []string
	fails             bool
	wantReview        bool
	missingAssessment bool
	noTestFiles       bool
	largeTestOutput   bool
}

func TestSecurityEvidenceRequiresEveryDeclaredRealCheck(t *testing.T) {
	for _, test := range securityEvidenceCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			apptest.RequireDocker(t)
			securityEvidenceScenario(t, test)
		})
	}
}

func securityEvidenceCases() []securityEvidenceCase {
	return []securityEvidenceCase{
		{
			name:     "all_named_checks_pass",
			commands: []string{"go test ./...", "go vet ./..."},
			declared: []string{"go test ./...", "go vet ./..."}, wantReview: true,
		},
		{
			name:     "one_of_two_named_checks_is_missing",
			commands: []string{"go test ./..."},
			declared: []string{"go test ./...", "go vet ./..."},
		},
		{
			name:     "failed_latest_check",
			commands: []string{"go test ./..."},
			declared: []string{"go test ./..."}, fails: true,
		},
		{
			name:     "test_listing_is_not_a_test_run",
			commands: []string{"go test -list ."},
			declared: []string{"go test -list ."},
		},
		{
			name:     "inspection_is_not_a_test_run",
			commands: []string{"pwd"}, declared: []string{"pwd"},
		},
		{
			name:     "go_test_dry_run_is_not_a_test_run",
			commands: []string{"go test -n ./..."}, declared: []string{"go test -n ./..."},
		},
		{
			name:     "go_test_compile_only_is_not_a_test_run",
			commands: []string{"go test -c -o /dev/null ./..."},
			declared: []string{"go test -c -o /dev/null ./..."},
		},
		{
			name:     "unmatched_selector_did_not_execute_tests",
			commands: []string{"go test -run UnmatchedSelector ./..."},
			declared: []string{"go test -run UnmatchedSelector ./..."},
		},
		{
			name:        "package_with_no_test_files_is_not_a_test_run",
			commands:    []string{"go test ./no-tests"},
			declared:    []string{"go test ./no-tests"},
			noTestFiles: true,
		},
		{
			name:            "truncated_go_test_output_fails_closed",
			commands:        []string{"go test -v ./..."},
			declared:        []string{"go test -v ./..."},
			largeTestOutput: true,
		},
		{
			name:              "otherwise_valid_check_without_assessment",
			commands:          []string{"go test ./..."},
			declared:          []string{"go test ./..."},
			missingAssessment: true,
		},
	}
}

func securityEvidenceScenario(t *testing.T, test securityEvidenceCase) {
	t.Helper()
	steps := securityEvidenceSteps(test)
	a := apptest.NewAgentRig(t, steps...)
	if test.noTestFiles {
		a.PutFile(t, "no-tests/package.go", "package notests\n")
	}
	configureReviewTeam(t, a, "developer", "security")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Update token handling","team":true,"planFirst":false}`,
		apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	want := "failed"
	if test.wantReview {
		want = "review"
	}
	a.WaitApproving(t, id, want)
	assertNoScriptExhaustion(t, a, id)
	assertEvidenceOutcome(t, a, id, test.wantReview)
}

func securityEvidenceSteps(test securityEvidenceCase) []string {
	steps := []string{
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: prepare a package", "state": "active"},
			{"title": "Security: run named checks", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "developer", "brief": "Prepare the package"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "developer", "brief": "Prepare the package"}),
		apptest.ToolCall("m", "write_file", map[string]string{
			"path": "go.mod", "content": "module sample\n\ngo 1.22\n",
		}),
		apptest.ToolCall("src", "write_file", map[string]string{
			"path": "token.go", "content": "package sample\nfunc value() int { return 1 }\n",
		}),
		apptest.ToolCall("test", "write_file", map[string]string{
			"path": "token_test.go", "content": packageTest(test.fails, test.largeTestOutput),
		}),
		apptest.ToolCall("sensitive", "write_file", map[string]string{
			"path": "internal/modules/auth/token.go", "content": "package auth\n",
		}),
		apptest.ToolCall("d-report", "report", map[string]string{"summary": "Prepared the package"}),
		apptest.FinishCall("Prepared the package"),
		apptest.ToolCall("p2", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: prepare a package", "state": "done"},
			{"title": "Security: run named checks", "state": "pending"},
		}}),
		apptest.ToolCall("s", "delegate", map[string]string{"role": "security", "brief": "Run the named checks"}),
	}
	for index, command := range test.commands {
		steps = append(steps, checkToolCall(index, command))
	}
	lines := make([]string, 0, len(test.declared)+2)
	lines = append(lines, "Verdict: PASS", "Checks:")
	for _, command := range test.declared {
		lines = append(lines, "- "+command)
	}
	report := strings.Join(lines, "\n")
	steps = append(steps, apptest.ToolCall("s-report", "report", map[string]string{
		"summary": securityReport(report, evidenceMatrixPaths(test.missingAssessment), test.commands),
	}))
	for range 8 {
		steps = append(steps, apptest.FinishCall("The checks passed"))
	}
	return steps
}

func checkToolCall(index int, command string) string {
	return apptest.ToolCall("check"+strconv.Itoa(index), "run_command", map[string]string{
		"command": command,
	})
}

func assertEvidenceOutcome(t *testing.T, a apptest.AgentRig, id string, wantReview bool) {
	t.Helper()
	if wantReview {
		events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
			apptest.JSONCookie(a.Cookie)).Body)
		if strings.Contains(events, `"kind":"error"`) {
			t.Fatal("all recorded named checks passed but the review was rejected")
		}
		return
	}
	assertBlockedTask(t, a, id, "token.go")
}

func assessmentPaths(missing bool) []string {
	if missing {
		return nil
	}
	return []string{"internal/modules/auth/token.go"}
}

func evidenceMatrixPaths(missing bool) []string {
	if missing {
		return nil
	}
	return []string{"go.mod", "token.go", "token_test.go", "internal/modules/auth/token.go"}
}

func securityReport(report string, paths, commands []string) string {
	if paths == nil {
		return report
	}
	assessment := "Assessment: " + strings.Join(paths, ", ") +
		" are the affected paths; their changed behavior is assessed together as the token boundary."
	if len(commands) > 0 {
		assessment += " Recorded scenarios " + strings.Join(commands, ", ") + " assess that boundary."
	}
	return report + "\n" + assessment
}

func packageTest(fails, largeOutput bool) string {
	value := "1"
	if fails {
		value = "2"
	}
	imports := "import \"testing\""
	log := ""
	if largeOutput {
		imports = "import (\n\t\"strings\"\n\t\"testing\"\n)"
		log = "t.Log(strings.Repeat(\"x\", 70000)); "
	}
	return "package sample\n\n" + imports + "\n\n" +
		"func TestValue(t *testing.T) { " + log +
		"if value() != " + value + " { t.Fatal(\"wrong value\") } }\n"
}

func TestUnclassifiedPathUsesConservativeMandatoryReviews(t *testing.T) {
	t.Parallel()
	assertBlockedForImpact(t, "vendor/custom-format.dat", "Update the data", "unclassified impact",
		"vendor/custom-format.dat")
}

func assertBlockedForImpact(t *testing.T, path, request, requiredText, requiredPath string) {
	t.Helper()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("write", "write_file", map[string]string{"path": path, "content": "updated"}),
		apptest.FinishCall("Updated the data"),
	)
	id := a.Start(t, request)
	a.WaitApproving(t, id, "failed")
	assertNoScriptExhaustion(t, a, id)

	for _, event := range a.Thread(t, id) {
		if event.Kind == "error" && strings.Contains(event.Text, requiredText) &&
			strings.Contains(event.Text, requiredPath) {
			return
		}
	}
	t.Fatalf("blocked task did not explain %q for %s", requiredText, path)
}
