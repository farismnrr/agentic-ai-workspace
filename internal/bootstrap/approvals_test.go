package bootstrap

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func skipApprovals(t *testing.T, app *fiber.App, cookie string) {
	t.Helper()

	if got := do(t, app, "PUT", "/api/spaces/coding/approvals", `{"skip":true}`, jsonCookie(cookie)).Status; got != 204 {
		t.Fatalf("saving the setting failed: %d", got)
	}
}

func TestSkipApprovalsIsPerSpaceAndNeverTurnsAToolOn(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, _ := signedInClient(t, app)

	if got := do(t, app, "PUT", "/api/spaces/nowhere/approvals", `{"skip":true}`, jsonCookie(cookie)).Status; got != 404 {
		t.Fatalf("an unknown space has no approvals setting, got %d", got)
	}

	mustContain(t, do(t, app, "GET", "/api/spaces", "", jsonCookie(cookie)).Body, `"skipApprovals":false`)

	skipApprovals(t, app, cookie)

	var spaces []struct {
		Tools []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"tools"`
		SkipApprovals bool `json:"skipApprovals"`
	}

	if err := json.Unmarshal(do(t, app, "GET", "/api/spaces", "", jsonCookie(cookie)).Body, &spaces); err != nil {
		t.Fatal(err)
	}

	if len(spaces) != 1 || !spaces[0].SkipApprovals {
		t.Fatalf("the setting should be saved for the space: %+v", spaces)
	}

	for _, tool := range spaces[0].Tools {
		if tool.Name == "edit_file" && tool.Enabled {
			t.Fatal("skipping approvals must never switch a tool on")
		}
	}
}

func TestAgentRunsEnabledToolsUnaskedWhenApprovalsAreSkipped(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		plan(),
		toolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		finishCall("Made hello.txt"),
	)

	skipApprovals(t, a.env.app, a.cookie)

	a.waitFor(t, a.start(t, "Make a hello file"), "review")

	if a.read("hello.txt") != "hi" {
		t.Fatal("with approvals skipped the enabled tool runs without asking")
	}
}

func TestAgentCannotUseAToolThatIsSwitchedOffEvenWhenApprovalsAreSkipped(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		plan(),
		toolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		finishCall("Tried"),
	)
	headers := jsonCookie(a.cookie)

	do(t, a.env.app, "PUT", "/api/spaces/coding/tools/write_file", `{"enabled":false}`, headers)
	skipApprovals(t, a.env.app, a.cookie)

	id := a.start(t, "Make a hello file")
	a.waitFor(t, id, "done")

	if a.read("hello.txt") != "" {
		t.Fatal("a switched-off tool must not run, bypass or not")
	}

	a.gateway.mu.Lock()
	offered, _ := json.Marshal(a.gateway.requests[0]["tools"])
	a.gateway.mu.Unlock()

	if strings.Contains(string(offered), `"write_file"`) {
		t.Fatal("a switched-off tool must not even be offered to the model")
	}
}

func TestConnectedAppCannotStartASiteUntilItIsSwitchedOn(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read coding:exec")

	off := callTool(t, app, token, "start_site", `{"command":"true","port":8123}`)
	mustContain(t, off.Body, `"isError":true`, "turned off")
}

func TestConnectedAppStartsASiteInItsProject(t *testing.T) {
	t.Parallel()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	env := newEnvWith(t, func(c *config.Config) { c.SandboxImage = image })
	cookie, clientID := signedInClient(t, env.app)
	token := connect(t, env.app, cookie, clientID, "coding:read coding:exec")
	enableTool(t, env.app, cookie, "start_site")
	enableTool(t, env.app, cookie, "stop_site")

	args, _ := json.Marshal(map[string]any{"command": siteCommand, "port": 8124})
	started := callTool(t, env.app, token, "start_site", string(args))
	mustContain(t, started.Body, publicURL+`/run/chatgpt/`, `\"running\":true`)

	mustContain(t, callTool(t, env.app, token, "stop_site", `{}`).Body, "stopped")
}
