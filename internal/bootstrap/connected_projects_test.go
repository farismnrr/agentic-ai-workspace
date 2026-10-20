package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAConnectedAppWorksInAnyProjectByItsProjectID(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, chatgpt := signedInClient(t, env.app)

	created := do(t, env.app, "POST", "/api/projects", `{"name":"Sensio","id":"sensio"}`, jsonCookie(cookie))
	if created.Status != 201 {
		t.Fatalf("create failed: %d %s", created.Status, created.Body)
	}

	uuid, _ := created.JSON(t)["id"].(string)
	file := filepath.Join(env.dataDir, "projects", uuid, "hello.txt")
	if err := os.WriteFile(file, []byte("hi from sensio"), 0o600); err != nil {
		t.Fatal(err)
	}

	token := connect(t, env.app, cookie, chatgpt, "coding:read")

	listed := callTool(t, env.app, token, "list_projects", `{}`)
	mustContain(t, listed.Body, `\"id\":\"sensio\"`, `\"default\":true`, "ChatGPT")

	named := callTool(t, env.app, token, "read_file", `{"path":"hello.txt","project":"sensio"}`)
	mustContain(t, named.Body, "hi from sensio")

	// Without a project the call stays in the app's own project, which has no such file.
	own := callTool(t, env.app, token, "read_file", `{"path":"hello.txt"}`)
	if strings.Contains(string(own.Body), "hi from sensio") {
		t.Fatalf("a call that names no project must stay in the app's own project: %s", own.Body)
	}

	unknown := callTool(t, env.app, token, "read_file", `{"path":"hello.txt","project":"nope"}`)
	mustContain(t, unknown.Body, "list_projects")
}

func TestAConnectedAppsWorkShowsInItsThreadWhileItRuns(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, chatgpt := signedInClient(t, env.app)
	enableTool(t, env.app, cookie, "run_command")

	token := connect(t, env.app, cookie, chatgpt, "coding:read coding:write coding:exec")

	init := do(t, env.app, "POST", "/api/coding", rpc("initialize",
		`{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}`), mcpHeaders(token))
	mustContain(t, init.Body, "log_message", "list_projects")

	callTool(t, env.app, token, "list_files", `{}`)
	callTool(t, env.app, token, "run_command", `{"command":"echo hello-from-the-app"}`)

	var tasks []struct {
		ID     string `json:"id"`
		Source string `json:"source"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(do(t, env.app, "GET", "/api/tasks", "", jsonCookie(cookie)).Body, &tasks); err != nil {
		t.Fatal(err)
	}

	var id string

	for _, task := range tasks {
		if task.Source == "ChatGPT" {
			id = task.ID

			if task.Status != "done" {
				t.Fatalf("the thread is done again once the call ended, got %s", task.Status)
			}
		}
	}

	if id == "" {
		t.Fatalf("the app's work must be in a thread: %+v", tasks)
	}

	events := string(do(t, env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(cookie)).Body)
	mustContain(t, []byte(events), "list .", "Running: command: echo hello-from-the-app")
	mustContain(t, []byte(events), "command: echo hello-from-the-app")
}
