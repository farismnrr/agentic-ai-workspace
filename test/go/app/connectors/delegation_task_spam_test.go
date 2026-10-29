package connectors_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

type delegatedBegin struct {
	Result struct {
		Structured struct {
			WorkID  string `json:"work_id"`
			TaskID  string `json:"task_id"`
			Resumed bool   `json:"resumed"`
		} `json:"structuredContent"`
	} `json:"result"`
}

func mcpBegin(t *testing.T, app *apptest.Environment, token, args string) delegatedBegin {
	t.Helper()
	call := apptest.CallTool(t, app.App, token, "begin_work", args)
	var parsed delegatedBegin
	if err := json.Unmarshal(call.Body, &parsed); err != nil || parsed.Result.Structured.WorkID == "" {
		t.Fatalf("begin_work: err=%v response=%s", err, call.Body)
	}
	return parsed
}

func visibleTasks(t *testing.T, env apptest.Environment, cookie string) []struct {
	ID     string `json:"id"`
	Origin string `json:"originTaskId"`
} {
	t.Helper()
	res := apptest.Do(t, env.App, "GET", "/api/tasks", "", apptest.JSONCookie(cookie))
	var out []struct {
		ID     string `json:"id"`
		Origin string `json:"originTaskId"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil {
		t.Fatalf("task list: %v %s", err, res.Body)
	}
	return out
}

func TestConnectedRoleDelegationsKeepOneVisibleTask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	one := mcpBegin(t, &env, token, `{"summary":"Fix task list duplication","role":"planner"}`)
	two := mcpBegin(t, &env, token, `{"summary":"Fix task list duplication","role":"researcher"}`)
	if one.Result.Structured.WorkID != two.Result.Structured.WorkID ||
		one.Result.Structured.TaskID != two.Result.Structured.TaskID || !two.Result.Structured.Resumed {
		t.Fatalf("delegated role created another top-level task: first=%+v second=%+v", one.Result.Structured, two.Result.Structured)
	}
	role := apptest.CallTool(t, env.App, token, "set_role", `{"work_id":"`+one.Result.Structured.WorkID+`","role":"developer"}`)
	if role.Status != 200 {
		t.Fatalf("set role: %d %s", role.Status, role.Body)
	}
	tasks := visibleTasks(t, env, cookie)
	if len(tasks) != 1 || tasks[0].ID != one.Result.Structured.TaskID {
		t.Fatalf("one issue should show one task: %+v", tasks)
	}
	distinct := mcpBegin(t, &env, token, `{"summary":"Implement an unrelated feature","role":"developer","new_task":true}`)
	if distinct.Result.Structured.TaskID == one.Result.Structured.TaskID || len(visibleTasks(t, env, cookie)) != 2 {
		t.Fatal("explicit separate user issue was merged into prior task")
	}
}

func TestExternalLinkedChatStaysUnderOneVisibleTask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	parent := mcpBegin(t, &env, token, `{"summary":"Build one thing","role":"planner"}`).Result.Structured.TaskID
	gateway := apptest.NewGateway(t, apptest.FinishCall("Finished child implementation"))
	added := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k")
	provider, _ := added.JSON(t)["id"].(string)
	apptest.Do(t, env.App, "PUT", "/api/preferences", `{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`, apptest.JSONCookie(cookie))
	result := apptest.Do(t, env.App, "POST", "/api/tasks/"+parent+"/chat", `{"text":"Implement approved change"}`, apptest.JSONCookie(cookie))
	if result.Status != 200 {
		t.Fatalf("linked chat failed: %d %s", result.Status, result.Body)
	}
	linked, _ := result.JSON(t)["linkedTaskId"].(string)
	if linked == "" || linked == parent {
		t.Fatalf("linked child missing: %s", result.Body)
	}
	detail := apptest.Do(t, env.App, "GET", "/api/tasks/"+linked, "", apptest.JSONCookie(cookie))
	if detail.Status != 200 {
		t.Fatalf("linked child must stay accessible: %d", detail.Status)
	}
	if detail.JSON(t)["originTaskId"] != parent {
		t.Fatalf("linked parent missing: %s", detail.Body)
	}
	tasks := visibleTasks(t, env, cookie)
	if len(tasks) != 1 || tasks[0].ID != parent {
		t.Fatalf("linked internal task leaked into top-level list: %+v", tasks)
	}
}

func TestTrackedIssuesRemainIsolatedAcrossClientsAndProjects(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, chatgpt := apptest.SignedInClient(t, env.App)
	gemini := registerClient(t, env.App, "Gemini")
	chatToken := apptest.Connect(t, env.App, cookie, chatgpt, "coding:read")
	gemToken := apptest.Connect(t, env.App, cookie, gemini, "coding:read")
	oneProject := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Project One","id":"project-one"}`, apptest.JSONCookie(cookie))
	twoProject := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Project Two","id":"project-two"}`, apptest.JSONCookie(cookie))
	p1 := "project-one"
	p2 := "project-two"
	if p1 == "" || p2 == "" {
		t.Fatalf("project create: %s %s", oneProject.Body, twoProject.Body)
	}
	a := mcpBegin(t, &env, chatToken, `{"summary":"Issue one","role":"developer","project":"`+p1+`"}`)
	b := mcpBegin(t, &env, chatToken, `{"summary":"Issue in other project","role":"developer","project":"`+p2+`"}`)
	c := mcpBegin(t, &env, gemToken, `{"summary":"Gemini independent work","role":"developer","project":"`+p1+`"}`)
	if a.Result.Structured.WorkID == b.Result.Structured.WorkID || a.Result.Structured.WorkID == c.Result.Structured.WorkID || b.Result.Structured.WorkID == c.Result.Structured.WorkID {
		t.Fatalf("distinct projects/clients were merged: %s %s %s", a.Result.Structured.WorkID, b.Result.Structured.WorkID, c.Result.Structured.WorkID)
	}
	repeated := mcpBegin(t, &env, chatToken, `{"summary":"Issue one with a new role","role":"reviewer","project":"`+p1+`"}`)
	if repeated.Result.Structured.WorkID != a.Result.Structured.WorkID {
		t.Fatal("changing role created another issue")
	}
	if len(visibleTasks(t, env, cookie)) != 3 {
		t.Fatalf("expected 3 independent tasks, got %+v", visibleTasks(t, env, cookie))
	}
}

func TestCompletingTrackedIssueAllowsNextTopLevelTask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	first := mcpBegin(t, &env, token, `{"summary":"Complete the first issue","role":"developer"}`)
	done := apptest.CallTool(t, env.App, token, "complete_work", `{"work_id":"`+first.Result.Structured.WorkID+`","outcome":"completed","summary":"First issue done"}`)
	if done.Status != 200 {
		t.Fatalf("complete failed: %d %s", done.Status, done.Body)
	}
	second := mcpBegin(t, &env, token, `{"summary":"Start next issue","role":"researcher"}`)
	if second.Result.Structured.TaskID == first.Result.Structured.TaskID || second.Result.Structured.Resumed || len(visibleTasks(t, env, cookie)) != 2 {
		t.Fatalf("closed task reused instead of new issue: first=%+v second=%+v", first.Result.Structured, second.Result.Structured)
	}
}

func TestRoleDelegationWithNewBeginKeyStaysOnActiveIssue(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	first := mcpBegin(t, &env, token, `{"summary":"Fix the single issue","role":"planner","idempotency_key":"issue-one-plan-request"}`)
	repeated := mcpBegin(t, &env, token, `{"summary":"Review the same single issue","role":"reviewer","idempotency_key":"issue-one-review-request"}`)
	if first.Result.Structured.WorkID != repeated.Result.Structured.WorkID || len(visibleTasks(t, env, cookie)) != 1 {
		t.Fatalf("different role-specific begin keys split the same active issue: first=%+v second=%+v", first.Result.Structured, repeated.Result.Structured)
	}
	independent := mcpBegin(t, &env, token, `{"summary":"Another explicit issue","role":"developer","new_task":true,"idempotency_key":"separate-request-key"}`)
	if independent.Result.Structured.WorkID == first.Result.Structured.WorkID || len(visibleTasks(t, env, cookie)) != 2 {
		t.Fatal("explicit new task with idempotency key was merged")
	}
	retry := mcpBegin(t, &env, token, `{"summary":"Another explicit issue","role":"developer","new_task":true,"idempotency_key":"separate-request-key"}`)
	if !retry.Result.Structured.Resumed || retry.Result.Structured.WorkID != independent.Result.Structured.WorkID || len(visibleTasks(t, env, cookie)) != 2 {
		t.Fatal("explicit new task idempotent replay created duplicate")
	}
}

func TestResumedKeyRemainsIdempotentAfterCompletion(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	a := mcpBegin(t, &env, token, `{"summary":"One issue","role":"planner","idempotency_key":"root-issue-initial-key"}`)
	b := mcpBegin(t, &env, token, `{"summary":"Review same issue","role":"reviewer","idempotency_key":"root-issue-review-key"}`)
	if a.Result.Structured.WorkID != b.Result.Structured.WorkID {
		t.Fatal("role changed work")
	}
	ended := apptest.CallTool(t, env.App, token, "complete_work", `{"work_id":"`+a.Result.Structured.WorkID+`","outcome":"completed","summary":"Done"}`)
	if ended.Status != 200 {
		t.Fatalf("complete failed: %s", ended.Body)
	}
	// A fresh OAuth connection must resolve the persisted alias, not start over.
	reconnected := apptest.Connect(t, env.App, cookie, client, "coding:read")
	replay := mcpBegin(t, &env, reconnected, `{"summary":"Review same issue","role":"reviewer","idempotency_key":"root-issue-review-key"}`)
	if replay.Result.Structured.WorkID != a.Result.Structured.WorkID || !replay.Result.Structured.Resumed || len(visibleTasks(t, env, cookie)) != 1 {
		t.Fatalf("request key lost its idempotency after closure: %+v", replay.Result.Structured)
	}
	different := apptest.CallTool(t, env.App, token, "begin_work", `{"summary":"Different content with same key","role":"reviewer","idempotency_key":"root-issue-review-key"}`)
	if !strings.Contains(string(different.Body), `"isError":true`) {
		t.Fatalf("reused key with different body was accepted: %s", different.Body)
	}
}

func TestArchivedWorkCannotCaptureNewIssue(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read")
	a := mcpBegin(t, &env, token, `{"summary":"Archived issue","role":"developer"}`)
	result := apptest.Do(t, env.App, "POST", "/api/tasks/archive", `{"ids":["`+a.Result.Structured.TaskID+`"]}`, apptest.JSONCookie(cookie))
	if result.Status != 200 && result.Status != 204 {
		t.Fatalf("archive failed: %d %s", result.Status, result.Body)
	}
	b := mcpBegin(t, &env, token, `{"summary":"Another after archive","role":"developer"}`)
	if b.Result.Structured.TaskID == a.Result.Structured.TaskID || len(visibleTasks(t, env, cookie)) != 1 {
		t.Fatalf("archived work captured new issue: %+v", b.Result.Structured)
	}
}
