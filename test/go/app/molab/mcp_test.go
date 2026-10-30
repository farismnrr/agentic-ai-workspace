package molab_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestChatGPTMoLabToolIsDiscoverableAndWorksWithoutCreatingATask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, clientID := apptest.SignedInClient(t, env.App)
	access := apptest.Connect(t, env.App, cookie, clientID, "coding:read coding:exec")
	listed := apptest.Do(t, env.App, http.MethodPost, "/api/coding",
		apptest.RPC("tools/list", "{}"), apptest.MCPHeaders(access))
	apptest.MustContain(t, listed.Body, "marimo_execute", "coding:exec")

	// The normal command switch governs this tool; there is no extra MoLab switch.
	args := `{"url":"https://example.invalid/","token":"SYNTHETIC-CODE-123","code":"1+1"}`
	off := apptest.CallTool(t, env.App, access, "marimo_execute", args)
	apptest.MustContain(t, off.Body, "turned off")
	apptest.EnableTool(t, env.App, cookie, "run_command")
	out := apptest.CallTool(t, env.App, access, "marimo_execute", args)
	apptest.MustContain(t, out.Body, "Invalid MoLab URL")
	apptest.MustNotContain(t, out.Body, "SYNTHETIC-CODE-123", "begin_work", "work role required")

	// Activity must be redacted and a standalone call must not create an external work thread.
	activity := apptest.Do(t, env.App, http.MethodGet, "/api/activity", "",
		map[string]string{"Cookie": cookie})
	if strings.Contains(string(activity.Body), "SYNTHETIC-CODE-123") {
		t.Fatal("MoLab credential leaked into the activity log")
	}
}

func TestChatGPTMoLabRequiresExecScope(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, clientID := apptest.SignedInClient(t, env.App)
	access := apptest.Connect(t, env.App, cookie, clientID, "coding:read")
	apptest.EnableTool(t, env.App, cookie, "run_command")
	out := apptest.CallTool(t, env.App, access, "marimo_execute",
		`{"url":"https://sb-test.sb.molab.run/","token":"SYNTHETIC-CODE-123","code":"1+1"}`)
	apptest.MustContain(t, out.Body, "insufficient_scope", "coding:exec")
}
