package review_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func skipApprovals(t *testing.T, app *fiber.App, cookie string) {
	t.Helper()

	if got := apptest.Do(t, app, "PUT", "/api/spaces/coding/approvals", `{"skip":true}`,
		apptest.JSONCookie(cookie)).Status; got != 204 {
		t.Fatalf("saving the setting failed: %d", got)
	}
}

func TestSkipApprovalsIsPerSpaceAndNeverTurnsAToolOn(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	cookie, _ := apptest.SignedInClient(t, app)

	if got := apptest.Do(t, app, "PUT", "/api/spaces/nowhere/approvals", `{"skip":true}`,
		apptest.JSONCookie(cookie)).Status; got != 404 {
		t.Fatalf("an unknown space has no approvals setting, got %d", got)
	}

	apptest.MustContain(t, apptest.Do(t, app, "GET", "/api/spaces", "", apptest.JSONCookie(cookie)).Body,
		`"skipApprovals":false`)

	skipApprovals(t, app, cookie)

	var spaces []struct {
		Tools []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"tools"`
		SkipApprovals bool `json:"skipApprovals"`
	}

	if err := json.Unmarshal(apptest.Do(t, app, "GET", "/api/spaces", "", apptest.JSONCookie(cookie)).Body,
		&spaces); err != nil {
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

	a := apptest.NewAgentRig(t,
		apptest.Plan(),
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		apptest.FinishCall("Made hello.txt"),
	)

	skipApprovals(t, a.Environment.App, a.Cookie)

	a.WaitFor(t, a.Start(t, "Make a hello file"), "review")

	if a.Read("hello.txt") != "hi" {
		t.Fatal("with approvals skipped the enabled tool runs without asking")
	}
}

func TestAgentCannotUseAToolThatIsSwitchedOffEvenWhenApprovalsAreSkipped(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t,
		apptest.Plan(),
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		apptest.FinishCall("Tried"),
	)
	headers := apptest.JSONCookie(a.Cookie)

	apptest.Do(t, a.Environment.App, "PUT", "/api/spaces/coding/tools/write_file", `{"enabled":false}`, headers)
	skipApprovals(t, a.Environment.App, a.Cookie)

	id := a.Start(t, "Make a hello file")
	a.WaitFor(t, id, "done")

	if a.Read("hello.txt") != "" {
		t.Fatal("a switched-off tool must not run, bypass or not")
	}

	a.Gateway.Mu.Lock()
	offered, _ := json.Marshal(a.Gateway.Requests[0]["tools"])
	a.Gateway.Mu.Unlock()

	if strings.Contains(string(offered), `"write_file"`) {
		t.Fatal("a switched-off tool must not even be offered to the model")
	}
}

func TestConnectedAppCannotStartASiteUntilRunACommandIsSwitchedOn(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	cookie, clientID := apptest.SignedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "coding:read coding:exec")

	off := apptest.CallTool(t, app, token, "start_site", `{"command":"true","port":8123}`)
	apptest.MustContain(t, off.Body, `"isError":true`, "turned off")
}

func TestConnectedAppStartsASiteInItsProject(t *testing.T) {
	t.Parallel()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	env := apptest.NewEnvWith(t, func(c *config.Config) { c.SandboxImage = image })
	cookie, clientID := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, clientID, "coding:read coding:exec")
	apptest.EnableTool(t, env.App, cookie, "run_command")

	args, _ := json.Marshal(map[string]any{"command": apptest.SiteCommand, "port": 8124})
	started := apptest.CallTool(t, env.App, token, "start_site", string(args))
	apptest.MustContain(t, started.Body, apptest.PublicURL+`/run/chatgpt/`, `\"running\":true`)

	apptest.MustContain(t, apptest.CallTool(t, env.App, token, "stop_site", `{}`).Body, "stopped")
}
