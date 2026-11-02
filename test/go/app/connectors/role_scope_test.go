package connectors_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestExternalRolesEnforceToolsAndWorkflowOnServer(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	for _, name := range []string{"write_file", "edit_file", "run_command", "run_task", "verify", "run_build", "run_artifact", "git", "gh"} {
		apptest.EnableTool(t, env.App, cookie, name)
	}
	access := apptest.Connect(t, env.App, cookie, client, "coding:read coding:write coding:exec")
	// Only the owner can classify a named task; the model cannot declare it.
	root := apptest.UserWorkspace(t, env, cookie, "ChatGPT")
	taskConfig := filepath.Join(root, ".workspace", "tasks.json")
	if err := os.MkdirAll(filepath.Dir(taskConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskConfig, []byte(`{"tasks":{
		"check":{"kind":"check","description":"QA check","command":["echo","check"]},
		"build":{"kind":"build","description":"DevOps build","command":["echo","build"]},
		"artifact":{"kind":"artifact","description":"Developer document/media","command":["echo","artifact"]}
	}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	one := mcpBegin(t, &env, access, `{"summary":"Enforce external team role scopes","role":"researcher"}`)
	workID := one.Result.Structured.WorkID
	call := func(tool, args string) string {
		t.Helper()
		return string(apptest.CallTool(t, env.App, access, tool, args).Body)
	}
	setRole := func(role string) {
		t.Helper()
		reply := call("set_role", `{"work_id":"`+workID+`","role":"`+role+`"}`)
		if !strings.Contains(reply, `"changed":true`) {
			t.Fatalf("cannot switch to %s: %s", role, reply)
		}
	}
	denied := func(tool, args string) {
		t.Helper()
		reply := call(tool, args)
		if !strings.Contains(reply, `"isError":true`) ||
			!strings.Contains(reply, "active role cannot use "+tool) {
			t.Fatalf("%s unexpectedly allowed: %s", tool, reply)
		}
	}

	read := call("list_files", `{"work_id":"`+workID+`"}`)
	if strings.Contains(read, `"isError":true`) {
		t.Fatalf("researcher could not read: %s", read)
	}
	file := filepath.Join(env.DataDir, "projects", "chatgpt", "role-scope.txt")
	writeArgs := `{"work_id":"` + workID + `","path":"role-scope.txt","content":"hello"}`
	denied("write_file", writeArgs)
	denied("run_command", `{"work_id":"`+workID+`","command":"echo no"}`)
	denied("run_task", `{"work_id":"`+workID+`","name":"test"}`)
	denied("verify", `{"work_id":"`+workID+`","args":["go","test","./..."]}`)
	denied("save_requirements", `{"work_id":"`+workID+`","markdown":"not my responsibility"}`)
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("researcher wrote file outside its role: %v", err)
	}

	setRole("planner")
	req := call("save_requirements", `{"work_id":"`+workID+`","markdown":"# Scope\nThe task must use real server-side roles."}`)
	if strings.Contains(req, `"isError":true`) {
		t.Fatalf("planner could not save requirements: %s", req)
	}
	denied("write_file", writeArgs)
	denied("run_command", `{"work_id":"`+workID+`","command":"echo no"}`)
	denied("gh", `{"work_id":"`+workID+`","args":["issue","view","1"]}`)

	setRole("developer")
	write := call("write_file", writeArgs)
	if strings.Contains(write, `"isError":true`) {
		t.Fatalf("developer could not edit: %s", write)
	}
	denied("run_task", `{"work_id":"`+workID+`","name":"test"}`)
	denied("run_command", `{"work_id":"`+workID+`","command":"go test ./..."}`)
	denied("verify", `{"work_id":"`+workID+`","args":["go","test","./..."]}`)
	gitResult := call("git", `{"work_id":"`+workID+`","args":[]}`)
	if strings.Contains(gitResult, "active role cannot use git") ||
		!strings.Contains(gitResult, "Git or GitHub CLI arguments") {
		t.Fatalf("Developer should own Git without shell, input must be validated: %s", gitResult)
	}
	missingGitHub := call("git", `{"work_id":"`+workID+`","args":["fetch","origin"]}`)
	if !strings.Contains(missingGitHub, "CONNECT_GITHUB_REQUIRED") ||
		!strings.Contains(missingGitHub, "No network request was made") {
		t.Fatalf("remote Git without the owner connection should stop with an action: %s", missingGitHub)
	}

	artifact := call("run_artifact", `{"work_id":"`+workID+`","name":"artifact"}`)
	if strings.Contains(artifact, "active role cannot use run_artifact") || strings.Contains(artifact, "unknown task") {
		t.Fatalf("Developer should reach the named artifact runner: %s", artifact)
	}

	ghDenied := call("gh", `{"work_id":"`+workID+`","args":["workflow","run","deploy.yml"]}`)
	if !strings.Contains(ghDenied, "Git or GitHub CLI arguments") {
		t.Fatalf("Developer must not start CI workflows: %s", ghDenied)
	}

	setRole("qa")
	denied("git", `{"work_id":"`+workID+`","args":["status"]}`)
	denied("run_artifact", `{"work_id":"`+workID+`","name":"artifact"}`)
	denied("write_file", writeArgs)
	denied("run_command", `{"work_id":"`+workID+`","command":"printf malicious > altered.txt"}`)
	verifyWithoutNamedTask := call("verify", `{"work_id":"`+workID+`","args":["bash","-c","go test ./..."]}`)
	if strings.Contains(verifyWithoutNamedTask, "active role cannot use verify") ||
		!strings.Contains(verifyWithoutNamedTask, "verification command must be") {
		t.Fatalf("QA should reach direct verification validation without named checks: %s", verifyWithoutNamedTask)
	}
	check := call("run_task", `{"work_id":"`+workID+`","name":"check"}`)
	if strings.Contains(check, "active role cannot use run_task") {
		t.Fatalf("QA should reach the named check runner: %s", check)
	}

	wrongKind := call("run_task", `{"work_id":"`+workID+`","name":"build"}`)
	if !strings.Contains(wrongKind, "unknown task") {
		t.Fatalf("QA could run a build task: %s", wrongKind)
	}

	// The legacy Reviewer key resolves to the existing QA role, not a new role.
	legacyReview := call("set_role", `{"work_id":"`+workID+`","role":"reviewer"}`)
	if !strings.Contains(legacyReview, `"role":{"key":"qa"`) {
		t.Fatalf("retired Reviewer was not merged into QA: %s", legacyReview)
	}
	denied("write_file", writeArgs)

	qaGhDenied := call("gh", `{"work_id":"`+workID+`","args":["issue","close","1"]}`)
	if !strings.Contains(qaGhDenied, "Git or GitHub CLI arguments") {
		t.Fatalf("QA must not mutate issues: %s", qaGhDenied)
	}

	setRole("devops")
	devopsGit := call("git", `{"work_id":"`+workID+`","args":["status"]}`)
	if strings.Contains(devopsGit, "active role cannot use git") {
		t.Fatalf("DevOps should receive bounded native Git: %s", devopsGit)
	}
	devopsMissingGitHub := call("git", `{"work_id":"`+workID+`","args":["fetch","origin"]}`)
	if !strings.Contains(devopsMissingGitHub, "CONNECT_GITHUB_REQUIRED") {
		t.Fatalf("DevOps remote Git without the owner connection should stop with an action: %s", devopsMissingGitHub)
	}
	denied("run_artifact", `{"work_id":"`+workID+`","name":"artifact"}`)
	denied("write_file", writeArgs)
	denied("run_task", `{"work_id":"`+workID+`","name":"echo"}`)
	denied("verify", `{"work_id":"`+workID+`","args":["go","test","./..."]}`)
	// DevOps recovery commands reach the read-only sandbox; this black-box
	// test only checks role authorization (Docker may be unavailable in CI).
	devopsCommand := call("run_command", `{"work_id":"`+workID+`","command":"printf devops-check"}`)
	if strings.Contains(devopsCommand, "active role cannot use run_command") {
		t.Fatalf("DevOps recovery execution must reach the sandbox: %s", devopsCommand)
	}
	build := call("run_build", `{"work_id":"`+workID+`","name":"build"}`)
	if strings.Contains(build, "active role cannot use run_build") || strings.Contains(build, "unknown task") {
		t.Fatalf("DevOps should reach its named build runner: %s", build)
	}
	devopsGhDenied := call("gh", `{"work_id":"`+workID+`","args":["pr","create"]}`)
	if !strings.Contains(devopsGhDenied, "Git or GitHub CLI arguments") {
		t.Fatalf("DevOps must not create source PRs: %s", devopsGhDenied)
	}
	wrongBuild := call("run_build", `{"work_id":"`+workID+`","name":"check"}`)
	if !strings.Contains(wrongBuild, "unknown task") {
		t.Fatalf("DevOps could run a QA check: %s", wrongBuild)
	}
	visible := visibleTasks(t, env, cookie)
	if len(visible) != 1 {
		t.Fatalf("role switches must remain in one task, got %d", len(visible))
	}
}
