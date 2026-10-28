package execution_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestOrchestratorFollowsChangedTierOnTheSameTask(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t, apptest.FinishCall("First answer"), apptest.FinishCall("Second answer"))
	headers := apptest.JSONCookie(a.Cookie)
	team := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", headers).JSON(t)
	roles, _ := team["roles"].([]any)
	for _, item := range roles {
		role, _ := item.(map[string]any)
		if role["key"] == "orchestrator" {
			role["tier"] = "thinking"
		}
	}
	body, _ := json.Marshal(team)
	if got := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", string(body), headers); got.Status != 200 {
		t.Fatal(string(got.Body))
	}
	prefs := apptest.Do(t, a.Environment.App, "GET", "/api/preferences", "", headers).JSON(t)
	provider, _ := prefs["defaultProviderId"].(string)
	saveThinking(t, a, prefs, provider, "tier-first")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks", `{"projectId":"`+a.Project+`","prompt":"Hello"}`,
		headers)
	id, _ := created.JSON(t)["id"].(string)
	a.WaitFor(t, id, "done")
	saveThinking(t, a, prefs, provider, "tier-second")
	follow := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages", `{"text":"Continue please"}`, headers)
	if follow.Status != 204 {
		t.Fatal(string(follow.Body))
	}
	a.WaitFor(t, id, "done")
	a.Gateway.Mu.Lock()
	defer a.Gateway.Mu.Unlock()
	if len(a.Gateway.Requests) != 2 || a.Gateway.Requests[0]["model"] != "tier-first" ||
		a.Gateway.Requests[1]["model"] != "tier-second" {
		t.Fatalf("actual models: %+v", a.Gateway.Requests)
	}
	apptest.MustContain(t, apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "", headers).Body,
		`"model":"tier-second"`, `"effort":"high"`)
	events := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers)
	apptest.MustContain(t, events.Body, "Orchestrator now uses tier-second")
}

func saveThinking(t *testing.T, a apptest.AgentRig, prefs map[string]any, provider, model string) {
	t.Helper()
	prefs["thinking"] = map[string]string{"providerId": provider, "model": model, "effort": "high"}
	body, _ := json.Marshal(prefs)
	if res := apptest.Do(t, a.Environment.App, "PUT", "/api/preferences", string(body),
		apptest.JSONCookie(a.Cookie)); res.Status != 200 {
		t.Fatal(string(res.Body))
	}
}

func TestLongMessagesBecomeCompleteProjectDocumentsOnCreateAndFollowUp(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t, apptest.FinishCall("Read the document"), apptest.FinishCall("Read the next document"))
	text := "  Start\n" + strings.Repeat("猫🙂resume ", 15_000) + "\nEND-OF-USER-REQUEST  "
	body, _ := json.Marshal(map[string]any{"projectId": a.Project, "prompt": text, "team": false})
	res := apptest.Do(t, a.Environment.App, "POST", "/api/tasks", string(body), apptest.JSONCookie(a.Cookie))
	if res.Status != 201 {
		t.Fatalf("long create: %d %s", res.Status, res.Body)
	}
	id, _ := res.JSON(t)["id"].(string)
	a.WaitFor(t, id, "done")
	checkOriginalInput(t, a, id, text)
	events := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", apptest.JSONCookie(a.Cookie))
	var thread []struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(events.Body, &thread); err != nil {
		t.Fatal(err)
	}
	if thread[0].Kind != "request" || !strings.HasPrefix(thread[0].Text, "Long message saved") {
		t.Fatal("the user's bubble must show a document preview rather than internal model instructions")
	}
	body, _ = json.Marshal(map[string]string{"text": text + "FOLLOW-UP-END"})
	res = apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages", string(body),
		apptest.JSONCookie(a.Cookie))
	if res.Status != 204 {
		t.Fatalf("long follow-up: %d %s", res.Status, res.Body)
	}
	a.WaitFor(t, id, "done")
	checkOriginalInput(t, a, id, text+"FOLLOW-UP-END")
}

func checkOriginalInput(t *testing.T, a apptest.AgentRig, id, original string) {
	t.Helper()
	var docs []struct {
		Path string `json:"path"`
	}
	res := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/documents", "", apptest.JSONCookie(a.Cookie))
	if err := json.Unmarshal(res.Body, &docs); err != nil || res.Status != 200 {
		t.Fatalf("documents: %d %s", res.Status, res.Body)
	}
	for _, doc := range docs {
		isOriginal := strings.HasSuffix(doc.Path, ".txt") && !strings.Contains(doc.Path, "/part-")
		if !strings.Contains(doc.Path, "/input/") || !isOriginal {
			continue
		}
		file := apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+a.Project+"/file/raw?path="+doc.Path, "",
			apptest.JSONCookie(a.Cookie))
		if file.Status == 200 && string(file.Body) == original {
			checkInputParts(t, a, strings.TrimSuffix(doc.Path, ".txt"), original)
			return
		}
	}
	t.Fatal("the exact original long message was not available through the project's file API")
}

func checkInputParts(t *testing.T, a apptest.AgentRig, base, original string) {
	t.Helper()
	var combined strings.Builder
	for part := 1; ; part++ {
		name := fmt.Sprintf("%s/part-%04d.txt", base, part)
		file := apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+a.Project+"/file?path="+name, "",
			apptest.JSONCookie(a.Cookie))
		if file.Status == 404 {
			break
		}
		content, _ := file.JSON(t)["content"].(string)
		if file.Status != 200 || content == "" {
			t.Fatalf("unreadable input part %s: %s", name, file.Body)
		}
		combined.WriteString(content)
	}
	if combined.String() != original {
		t.Fatal("reading every input part lost or changed content")
	}
}
