package connectors_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestAConnectedAppWorksInAnyProjectByItsProjectID(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, chatgpt := apptest.SignedInClient(t, env.App)

	created := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Sensio","id":"sensio"}`,
		apptest.JSONCookie(cookie))
	if created.Status != 201 {
		t.Fatalf("create failed: %d %s", created.Status, created.Body)
	}

	uuid, _ := created.JSON(t)["id"].(string)
	file := filepath.Join(env.DataDir, "projects", uuid, "hello.txt")
	if err := os.WriteFile(file, []byte("hi from sensio"), 0o600); err != nil {
		t.Fatal(err)
	}

	token := apptest.Connect(t, env.App, cookie, chatgpt, "coding:read")

	listed := apptest.CallTool(t, env.App, token, "list_projects", `{}`)
	apptest.MustContain(t, listed.Body, `\"id\":\"sensio\"`, `\"default\":true`, "ChatGPT")

	namedID := beginTrackedWork(t, env.App, token, "sensio")
	named := apptest.CallTool(t, env.App, token, "read_file",
		`{"work_id":"`+namedID+`","path":"hello.txt","project":"sensio"}`)
	apptest.MustContain(t, named.Body, "hi from sensio")

	// Without a project the call stays in the app's own project, which has no such file.
	ownID := beginTrackedWork(t, env.App, token, "")
	own := apptest.CallTool(t, env.App, token, "read_file", `{"work_id":"`+ownID+`","path":"hello.txt"}`)
	if strings.Contains(string(own.Body), "hi from sensio") {
		t.Fatalf("a call that names no project must stay in the app's own project: %s", own.Body)
	}

	unknown := apptest.CallTool(t, env.App, token, "read_file",
		`{"work_id":"`+ownID+`","path":"hello.txt","project":"nope"}`)
	apptest.MustContain(t, unknown.Body, "list_projects")
}

func TestAConnectedAppsWorkShowsInItsThreadWhileItRuns(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, chatgpt := apptest.SignedInClient(t, env.App)
	apptest.EnableTool(t, env.App, cookie, "run_command")

	token := apptest.Connect(t, env.App, cookie, chatgpt, "coding:read coding:write coding:exec")

	init := apptest.Do(t, env.App, "POST", "/api/coding", apptest.RPC("initialize",
		`{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}`),
		apptest.MCPHeaders(token))
	apptest.MustContain(t, init.Body, "log_message", "list_projects")

	workID := beginTrackedWork(t, env.App, token, "")
	apptest.CallTool(t, env.App, token, "list_files", `{"work_id":"`+workID+`"}`)
	// The Developer cannot run the shell. Switch role within the same task.
	switched := apptest.CallTool(t, env.App, token, "set_role",
		`{"work_id":"`+workID+`","role":"devops"}`)
	apptest.MustContain(t, switched.Body, `"changed":true`)
	apptest.CallTool(t, env.App, token, "run_command", `{"work_id":"`+workID+`","command":"echo hello-from-the-app"}`)

	var tasks []struct {
		ID     string `json:"id"`
		Source string `json:"source"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(apptest.Do(t, env.App, "GET", "/api/tasks", "", apptest.JSONCookie(cookie)).Body,
		&tasks); err != nil {
		t.Fatal(err)
	}

	var id string

	for _, task := range tasks {
		if task.Source == "ChatGPT" {
			id = task.ID

			if task.Status != "working" {
				t.Fatalf("tracked external work should remain open until completed, got %s", task.Status)
			}
		}
	}

	if id == "" {
		t.Fatalf("the app's work must be in a thread: %+v", tasks)
	}

	events := string(apptest.Do(t, env.App, "GET", "/api/tasks/"+id+"/events", "", apptest.JSONCookie(cookie)).Body)
	apptest.MustContain(t, []byte(events), "list .", `"role":"developer"`)
	apptest.MustContain(t, []byte(events), "command: echo hello-from-the-app")
}
