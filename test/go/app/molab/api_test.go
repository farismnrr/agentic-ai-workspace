package molab_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestMoLabPairingHTTPIsOwnerOnlyAndNeverReturnsTokens(t *testing.T) {
	rig := apptest.NewAgentRigWith(t, func(cfg *config.Config) { cfg.EgressGrants = true }, apptest.ToolCall("q1", "ask_user", map[string]any{
		"questions": []map[string]any{{"header": "Next", "question": "Ready for the notebook?", "type": "single",
			"options": []map[string]string{{"label": "Yes"}, {"label": "No"}}}},
	}))
	taskID := rig.Start(t, "Pair with a notebook")
	rig.WaitFor(t, taskID, "needs-input")
	path := "/api/tasks/" + taskID + "/molab"
	headers := apptest.JSONCookie(rig.Cookie)
	secret := "synthetic-secret-123"
	body := `{"url":"https://sb-example.sb.molab.run/","token":"` + secret + `"}`
	if got := apptest.Do(t, rig.Environment.App, "POST", path, body, nil).Status; got != http.StatusUnauthorized {
		t.Fatalf("anonymous token submission accepted: %d", got)
	}
	other := apptest.Register(t, rig.Environment.App, "Other", "other@example.test", "a sufficiently long password")
	if got := apptest.Do(t, rig.Environment.App, "POST", path, body, apptest.JSONCookie(other)).Status; got != http.StatusNotFound {
		t.Fatalf("other user modified pairing: %d", got)
	}
	cross := apptest.JSONCookie(rig.Cookie)
	cross["Origin"] = "https://attacker.invalid"
	if got := apptest.Do(t, rig.Environment.App, "POST", path, body, cross).Status; got != http.StatusForbidden {
		t.Fatalf("cross-site pairing accepted: %d", got)
	}
	created := apptest.Do(t, rig.Environment.App, "POST", path, body, headers)
	if created.Status != http.StatusOK {
		t.Fatalf("pair failed: %d %s", created.Status, created.Body)
	}
	if strings.Contains(string(created.Body), secret) {
		t.Fatal("POST echoed secret")
	}
	if created.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("POST response cacheable")
	}
	status := apptest.Do(t, rig.Environment.App, "GET", path, "", headers)
	if status.Status != http.StatusOK {
		t.Fatalf("GET status: %d", status.Status)
	}
	if strings.Contains(string(status.Body), secret) {
		t.Fatal("GET leaked secret")
	}
	if status.JSON(t)["connected"] != true {
		t.Fatal("GET did not show connection")
	}
	if status.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("GET response cacheable")
	}
	stopped := apptest.Do(t, rig.Environment.App, "DELETE", path, "", headers)
	if stopped.Status != http.StatusNoContent {
		t.Fatalf("disconnect status: %d", stopped.Status)
	}
	after := apptest.Do(t, rig.Environment.App, "GET", path, "", headers)
	if after.JSON(t)["connected"] != false {
		t.Fatal("disconnect retained pairing")
	}
}

func TestDisabledMoLabPairingNeverAcceptsSecret(t *testing.T) {
	rig := apptest.NewAgentRig(t, apptest.ToolCall("q1", "ask_user", map[string]any{
		"questions": []map[string]any{{"header": "Next", "question": "Wait for the user", "type": "single",
			"options": []map[string]string{{"label": "Yes"}, {"label": "No"}}}},
	}))
	id := rig.Start(t, "Review notebook")
	rig.WaitFor(t, id, "needs-input")
	path := "/api/tasks/" + id + "/molab"
	cookie := apptest.JSONCookie(rig.Cookie)
	state := apptest.Do(t, rig.Environment.App, "GET", path, "", cookie)
	if state.Status != http.StatusOK || state.JSON(t)["available"] != false {
		t.Fatal("disabled feature was exposed")
	}
	body := `{"url":"https://sb-example.sb.molab.run/","token":"synthetic-secret-123"}`
	forbidden := apptest.Do(t, rig.Environment.App, "POST", path, body, cookie)
	if forbidden.Status != http.StatusServiceUnavailable {
		t.Fatalf("disabled feature accepted secret: %d", forbidden.Status)
	}
	if strings.Contains(string(forbidden.Body), "synthetic-secret") {
		t.Fatal("disabled error echoed token")
	}
}
