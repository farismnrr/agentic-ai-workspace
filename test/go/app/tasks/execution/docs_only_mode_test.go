package execution_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestDeepReviewDocumentationOnlyChangeSkipsSecurityAndQA(t *testing.T) {
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("write-readme", "write_file", map[string]string{
			"path": "README.md", "content": "# Updated documentation\n",
		}),
		apptest.FinishCall("Updated readme"),
	)
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Correct documentation","mode":"deep-review","team":false}`,
		apptest.JSONCookie(a.Cookie))
	if created.Status != 201 {
		t.Fatalf("create: %d %s", created.Status, created.Body)
	}
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "review")
	events := a.Thread(t, id)
	for _, event := range events {
		if event.Kind == "error" && (strings.Contains(event.Text, "Security") || strings.Contains(event.Text, "QA")) {
			t.Fatalf("documentation-only edit should not require code review roles: %+v", event)
		}
		if event.Kind == "handoff" && (event.Agent == "Security" || event.Agent == "Architect") {
			t.Fatalf("unexpected docs-only reviewer: %+v", event)
		}
	}
}
