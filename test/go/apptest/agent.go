package apptest

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// AgentRig bundles a signed-in test person with a scripted model and project.
type AgentRig struct {
	Environment Environment
	Gateway     *Gateway
	Cookie      string
	Project     string
}

// NewAgentRig creates a person, model, and project for task integration tests.
func NewAgentRig(t *testing.T, steps ...string) AgentRig {
	t.Helper()
	return NewAgentRigWith(t, nil, steps...)
}

// NewAgentRigWith creates an agent rig with adjusted application configuration.
func NewAgentRigWith(t *testing.T, adjust func(*config.Config), steps ...string) AgentRig {
	t.Helper()
	return RigOn(t, NewEnvWith(t, adjust), steps...)
}

// RigOn adds a signed-in person and scripted model to an existing app.
func RigOn(t *testing.T, env Environment, steps ...string) AgentRig {
	t.Helper()
	cookie, _ := SignedInClient(t, env.App)
	gw := NewGateway(t, steps...)
	for _, tool := range []string{"edit_file", "write_file", "delete_file", "move_file", "run_task", "run_command"} {
		EnableTool(t, env.App, cookie, tool)
	}
	provider, _ := AddProvider(t, env, cookie, gw.Server.URL, "k").JSON(t)["id"].(string)
	Do(t, env.App, "POST", "/api/providers/"+provider+"/verify", "", JSONCookie(cookie))
	Do(t, env.App, "PUT", "/api/preferences",
		`{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`, JSONCookie(cookie))
	project, _ := Do(t, env.App, "POST", "/api/projects", `{"name":"Bakery"}`,
		JSONCookie(cookie)).JSON(t)["id"].(string)
	return AgentRig{Environment: env, Gateway: gw, Cookie: cookie, Project: project}
}

// Start creates a task using the rig's project.
func (a AgentRig) Start(t *testing.T, prompt string) string {
	t.Helper()
	response := Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"`+prompt+`","team":false}`, JSONCookie(a.Cookie))
	if response.Status != http.StatusCreated {
		t.Fatalf("task not created: %d %s", response.Status, response.Body)
	}
	id, _ := response.JSON(t)["id"].(string)
	return id
}

// Status returns the latest task status.
func (a AgentRig) Status(t *testing.T, id string) string {
	t.Helper()
	status, _ := Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "",
		JSONCookie(a.Cookie)).JSON(t)["status"].(string)
	return status
}

// WaitFor waits for a task to reach the requested status.
func (a AgentRig) WaitFor(t *testing.T, id, want string) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if a.Status(t, id) == want {
			return
		}
		time.Sleep(requestPollInterval)
	}
	t.Fatalf("task never reached %q, it is %q", want, a.Status(t, id))
}

// Post sends a JSON request using the rig's session.
func (a AgentRig) Post(t *testing.T, path, body string) int {
	t.Helper()
	return Do(t, a.Environment.App, "POST", path, body, JSONCookie(a.Cookie)).Status
}

// File returns a path inside the rig's project directory.
func (a AgentRig) File(name string) string {
	return filepath.Join(a.Environment.DataDir, "projects", a.Project, name)
}

// Read returns the contents of a file in the rig's project directory, or empty when absent.
func (a AgentRig) Read(name string) string {
	data, err := os.ReadFile(a.File(name))
	if err != nil {
		return ""
	}
	return string(data)
}

// PutChecks saves project checks through the API.
func (a AgentRig) PutChecks(t *testing.T, body string) Reply {
	t.Helper()
	return Do(t, a.Environment.App, "PUT", "/api/projects/"+a.Project+"/checks", body,
		JSONCookie(a.Cookie))
}

// Files sends a project-file request using the rig's session.
func (a AgentRig) Files(t *testing.T, method, route, body string) Reply {
	t.Helper()
	return Do(t, a.Environment.App, method, "/api/projects/"+a.Project+route, body,
		JSONCookie(a.Cookie))
}

// PutFile creates a file under the rig's isolated project folder.
func (a AgentRig) PutFile(t *testing.T, name, content string) {
	t.Helper()
	path := a.File(name)
	if err := os.MkdirAll(filepath.Dir(path), testDirectoryMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), testFileMode); err != nil {
		t.Fatal(err)
	}
}

// RequireDocker skips a test when Docker is unavailable.
func RequireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), commandTimeout)
	defer cancel()
	_, err := sandbox.Open(ctx, sandbox.Config{Image: "busybox:1.37", Limits: sandbox.DefaultLimits()})
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatalf("Docker is required for this CI scenario: %v", err)
		}
		t.Skipf("no Docker on this host: %v", err)
	}
}

// AddProvider registers a compatible model provider for a person.
func AddProvider(t *testing.T, env Environment, cookie, baseURL, key string) Reply {
	t.Helper()
	body := `{"kind":"compatible","name":"Local test gateway","baseUrl":"` + baseURL +
		`","apiKey":"` + key + `"}`
	return Do(t, env.App, "POST", "/api/providers", body, JSONCookie(cookie))
}

// EnableTool turns on one coding tool for a person.
func EnableTool(t *testing.T, app *fiber.App, cookie, tool string) {
	t.Helper()
	response := Do(t, app, "PUT", "/api/spaces/coding/tools/"+tool,
		`{"enabled":true}`, JSONCookie(cookie))
	if response.Status != http.StatusNoContent {
		t.Fatalf("enabling %s failed: %d", tool, response.Status)
	}
}
