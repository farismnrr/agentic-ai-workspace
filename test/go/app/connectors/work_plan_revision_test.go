package connectors_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestExternalPlanRevisionSurvivesRoleSwitch(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	workID := mcpBegin(t, &env, token, `{"summary":"Persist an external plan","role":"researcher"}`).Result.Structured.WorkID

	role := apptest.CallTool(t, env.App, token, "set_role",
		fmt.Sprintf(`{"work_id":%q,"role":"planner"}`, workID))
	if strings.Contains(string(role.Body), `"isError":true`) {
		t.Fatalf("switch to Planner failed: %s", role.Body)
	}

	readState := func() (int64, int64) {
		t.Helper()
		state := apptest.CallTool(t, env.App, token, "get_work_state",
			fmt.Sprintf(`{"work_id":%q}`, workID))
		if strings.Contains(string(state.Body), `"isError":true`) ||
			!strings.Contains(string(state.Body), `"plan_revision":`) {
			t.Fatalf("get_work_state must expose plan_revision: %s", state.Body)
		}
		var parsed struct {
			Result struct {
				Structured struct {
					Revision     int64 `json:"revision"`
					PlanRevision int64 `json:"plan_revision"`
				} `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(state.Body, &parsed); err != nil {
			t.Fatalf("decode state: %v: %s", err, state.Body)
		}
		return parsed.Result.Structured.Revision, parsed.Result.Structured.PlanRevision
	}

	workRevision, baseRevision := readState()
	if workRevision <= baseRevision || baseRevision != 0 {
		t.Fatalf("role change should advance work revision, not plan revision: work=%d plan=%d",
			workRevision, baseRevision)
	}
	update := apptest.CallTool(t, env.App, token, "update_plan",
		fmt.Sprintf(`{"work_id":%q,"base_revision":%d,"steps":[{"id":"one","title":"Implement fix","state":"pending"}]}`,
			workID, baseRevision))
	if strings.Contains(string(update.Body), `"isError":true`) {
		t.Fatalf("plan update using plan_revision failed: %s", update.Body)
	}

	workRevision, baseRevision = readState()
	if baseRevision != 1 || workRevision <= baseRevision {
		t.Fatalf("plan revision not independently persisted: work=%d plan=%d",
			workRevision, baseRevision)
	}
	stale := apptest.CallTool(t, env.App, token, "update_plan",
		fmt.Sprintf(`{"work_id":%q,"base_revision":%d,"steps":[{"id":"one","title":"Implement fix","state":"done"}]}`,
			workID, workRevision))
	if !strings.Contains(string(stale.Body), `"isError":true`) ||
		!strings.Contains(string(stale.Body), "work session changed") {
		t.Fatalf("using work revision must give actionable conflict, not internal error: %s", stale.Body)
	}
}
