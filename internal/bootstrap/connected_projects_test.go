package bootstrap

import (
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
