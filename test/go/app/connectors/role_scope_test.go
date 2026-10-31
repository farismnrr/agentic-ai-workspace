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
	for _, name := range []string{"write_file", "edit_file", "run_command", "run_task"} {
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
		"build":{"kind":"build","description":"DevOps build","command":["echo","build"]}
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

	setRole("developer")
	write := call("write_file", writeArgs)
	if strings.Contains(write, `"isError":true`) {
		t.Fatalf("developer could not edit: %s", write)
	}
	denied("run_task", `{"work_id":"`+workID+`","name":"test"}`)
	denied("run_command", `{"work_id":"`+workID+`","command":"go test ./..."}`)

	setRole("qa")
	denied("write_file", writeArgs)
	denied("run_command", `{"work_id":"`+workID+`","command":"printf malicious > altered.txt"}`)
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

	setRole("devops")
	denied("write_file", writeArgs)
	denied("run_task", `{"work_id":"`+workID+`","name":"echo"}`)
	denied("run_command", `{"work_id":"`+workID+`","command":"go test ./..."}`)
	build := call("run_build", `{"work_id":"`+workID+`","name":"build"}`)
	if strings.Contains(build, "active role cannot use run_build") || strings.Contains(build, "unknown task") {
		t.Fatalf("DevOps should reach its named build runner: %s", build)
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
