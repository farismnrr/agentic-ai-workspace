package execution_test

import (
	"encoding/json"
	"fmt"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func scriptedSummary(text string) string {
	body, _ := json.Marshal(map[string]any{"choices": []map[string]any{{"message": map[string]any{"content": text}}}, "usage": map[string]int{"prompt_tokens": 9000, "completion_tokens": 100}})
	return string(body)
}

func setContextLimit(t *testing.T, rig apptest.AgentRig, limit int) {
	t.Helper()
	providers := apptest.Do(t, rig.Environment.App, http.MethodGet, "/api/providers", "", apptest.JSONCookie(rig.Cookie))
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(providers.Body, &list); err != nil || len(list) == 0 {
		t.Fatalf("providers: %s err %v", providers.Body, err)
	}
	reply := apptest.Do(t, rig.Environment.App, http.MethodPut, "/api/providers/"+list[0].ID+"/context", fmt.Sprintf(`{"model":"fake-model","contextWindow":%d}`, limit), apptest.JSONCookie(rig.Cookie))
	if reply.Status != http.StatusNoContent {
		t.Fatalf("set context limit %d: %s", reply.Status, reply.Body)
	}
}

// Seed a persisted older conversation to reproduce an initially full *resumed*
// context. A giant new API prompt would be offloaded to a project document.
func resumeWithFullContext(t *testing.T, rig apptest.AgentRig) string {
	t.Helper()
	id := rig.Start(t, "Initial request: preserve files")
	rig.WaitFor(t, id, "done")
	setContextLimit(t, rig, 32000)
	block, err := json.Marshal(domain.Block{Kind: domain.BlockText, Text: "Older context: preserve files. " + strings.Repeat("Long transcript. ", 18000)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = rig.Environment.DB.Exec(`INSERT INTO task_transcript(task_id,seq,role,block,scope) VALUES(?,(SELECT COALESCE(MAX(seq),0)+1 FROM task_transcript WHERE task_id=?),'user',?,'')`, id, id, string(block)); err != nil {
		t.Fatal(err)
	}
	response := apptest.Do(t, rig.Environment.App, http.MethodPost, "/api/tasks/"+id+"/messages", `{"text":"Latest instruction: finish implementation"}`, apptest.JSONCookie(rig.Cookie))
	if response.Status != http.StatusNoContent {
		t.Fatalf("resume: %d %.300s", response.Status, response.Body)
	}
	return id
}

func waitModelCalls(t *testing.T, rig apptest.AgentRig, count int) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		rig.Gateway.Mu.Lock()
		got := len(rig.Gateway.Requests)
		rig.Gateway.Mu.Unlock()
		if got >= count {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("model never received %d calls (got %d)", count, len(rig.Gateway.Requests))
}

func TestInitiallyFullContextCompactsBeforeFirstNormalModelCall(t *testing.T) {
	rig := apptest.NewAgentRig(t, apptest.FinishCall("Initial answer"), scriptedSummary("Summary: preserve files, finish implementation"), apptest.Plan(), apptest.FinishCall("Done safely"))
	id := resumeWithFullContext(t, rig)
	rig.Gateway.Mu.Lock()
	t.Logf("initial calls: %d", len(rig.Gateway.Requests))
	rig.Gateway.Mu.Unlock()
	waitModelCalls(t, rig, 3)
	rig.Gateway.Mu.Lock()
	calls := append([]map[string]any(nil), rig.Gateway.Requests...)
	rig.Gateway.Mu.Unlock()
	first, _ := json.Marshal(calls[1]["messages"])
	second, _ := json.Marshal(calls[2]["messages"])
	if len(first) >= len(strings.Repeat("Long transcript. ", 18000))/2 {
		t.Fatalf("summary call was not shortened: %d bytes", len(first))
	}
	if !strings.Contains(string(first), "Initial request: preserve files") {
		t.Fatal("summary lost original instruction")
	}
	if calls[1]["tools"] != nil {
		t.Fatal("summary-only compaction must not offer executable tools")
	}
	if !strings.Contains(string(first), "Latest instruction: finish implementation") {
		t.Fatal("summary lost the latest resumed instruction")
	}
	if !strings.Contains(string(second), "Summary: preserve files") {
		t.Fatal("first normal model call missed compacted summary")
	}
	response := apptest.Do(t, rig.Environment.App, http.MethodGet, "/api/tasks/"+id, "", apptest.JSONCookie(rig.Cookie))
	var task struct {
		ContextUsed   int64  `json:"contextUsed"`
		ContextWindow int64  `json:"contextWindow"`
		ContextState  string `json:"contextState"`
	}
	if err := json.Unmarshal(response.Body, &task); err != nil {
		t.Fatal(err)
	}
	if task.ContextState != "measured" {
		t.Fatalf("compaction did not reset measured context: %+v", task)
	}
	if task.ContextWindow != 32000 {
		t.Fatalf("wrong context window: %+v", task)
	}
}

func TestInitiallyFullContextCompactionFailureStopsBeforeNormalCall(t *testing.T) {
	rig := apptest.NewAgentRig(t, apptest.FinishCall("Initial answer"), scriptedSummary(""))
	id := resumeWithFullContext(t, rig)
	rig.WaitFor(t, id, "failed")
	rig.Gateway.Mu.Lock()
	got := len(rig.Gateway.Requests)
	rig.Gateway.Mu.Unlock()
	if got != 2 {
		t.Fatalf("failed compaction must not submit normal oversized request: %d calls", got)
	}
	res := apptest.Do(t, rig.Environment.App, http.MethodGet, "/api/tasks/"+id, "", apptest.JSONCookie(rig.Cookie))
	if !strings.Contains(string(res.Body), `"contextState":"failed"`) {
		t.Fatalf("no durable failed context status: %.400s", res.Body)
	}
}
