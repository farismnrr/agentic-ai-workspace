package connectors_test

import (
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"net/http"
	"testing"
)

func TestManualStopExternalTaskIsDoneNotNeedsAttention(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	created := mcpBegin(t, &env, token, `{"summary":"Manual stop regression","role":"developer"}`)
	id := created.Result.Structured.TaskID
	res := apptest.Do(t, env.App, http.MethodPost, "/api/tasks/"+id+"/cancel", `{}`, apptest.JSONCookie(cookie))
	if res.Status != http.StatusNoContent {
		t.Fatalf("stop: %d %s", res.Status, res.Body)
	}
	task := apptest.Do(t, env.App, http.MethodGet, "/api/tasks/"+id, "", apptest.JSONCookie(cookie))
	data := task.JSON(t)
	if data["status"] != "done" {
		t.Fatalf("stopped task still needs attention: %s", task.Body)
	}
	work, _ := data["work"].(map[string]any)
	if work["state"] != "stopped" {
		t.Fatalf("lost intentional stop provenance: %s", task.Body)
	}
	if data["proof"] != nil {
		t.Fatalf("manual stop fabricated verified completion: %s", task.Body)
	}
	events := apptest.Do(t, env.App, http.MethodGet, "/api/tasks/"+id+"/events", "", apptest.JSONCookie(cookie))
	apptest.MustContain(t, events.Body, `"text":"Stopped by you"`, `"kind":"message"`)
	confirm := apptest.Do(t, env.App, http.MethodPost, "/api/tasks/"+id+"/confirm", `{}`, apptest.JSONCookie(cookie))
	if confirm.Status != http.StatusConflict {
		t.Fatalf("stopped work must not confirm: %d %s", confirm.Status, confirm.Body)
	}
	progress := apptest.CallTool(t, env.App, token, "report_progress", `{"work_id":"`+created.Result.Structured.WorkID+`","message":"late progress"}`)
	apptest.MustContain(t, progress.Body, `"isError":true`)
	again := apptest.Do(t, env.App, http.MethodGet, "/api/tasks/"+id, "", apptest.JSONCookie(cookie))
	if again.JSON(t)["status"] != "done" {
		t.Fatalf("late external activity reopened stopped task: %s", again.Body)
	}
}
