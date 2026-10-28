package connectors_test

import (
	"encoding/json"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"github.com/gofiber/fiber/v3"
)

// beginTrackedWork selects a real role before any task-producing MCP call.
func beginTrackedWork(t *testing.T, app *fiber.App, token, project string) string {
	t.Helper()
	args := `{"role":"developer","summary":"Exercise connected app work tracking"`
	if project != "" {
		args += `,"project":"` + project + `"`
	}
	begin := apptest.CallTool(t, app, token, "begin_work", args+"}")
	var response struct {
		Result struct {
			Structured struct {
				WorkID string `json:"work_id"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(begin.Body, &response); err != nil || response.Result.Structured.WorkID == "" {
		t.Fatalf("begin_work failed: %v %s", err, begin.Body)
	}
	return response.Result.Structured.WorkID
}
