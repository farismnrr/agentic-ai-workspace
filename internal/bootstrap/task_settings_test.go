package bootstrap

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestOrchestratorFollowsChangedTierOnTheSameTask(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t, finishCall("First answer"), finishCall("Second answer"))
	headers := jsonCookie(a.cookie)
	team := do(t, a.env.app, "GET", "/api/agent-team", "", headers).JSON(t)
	roles, _ := team["roles"].([]any)
	for _, item := range roles {
		role, _ := item.(map[string]any)
		if role["key"] == "orchestrator" {
			role["tier"] = "thinking"
		}
	}
	body, _ := json.Marshal(team)
	if got := do(t, a.env.app, "PUT", "/api/agent-team", string(body), headers); got.Status != 200 {
		t.Fatal(string(got.Body))
	}
	prefs := do(t, a.env.app, "GET", "/api/preferences", "", headers).JSON(t)
	provider, _ := prefs["defaultProviderId"].(string)
	saveThinking(t, a, prefs, provider, "tier-first")
	input, _ := json.Marshal(map[string]any{"projectId": a.project, "prompt": "Hello", "team": true})
	created := do(t, a.env.app, "POST", "/api/tasks", string(input), headers)
	id, _ := created.JSON(t)["id"].(string)
	a.waitFor(t, id, "done")
	saveThinking(t, a, prefs, provider, "tier-second")
	follow := do(t, a.env.app, "POST", "/api/tasks/"+id+"/messages", `{"text":"Continue please"}`, headers)
	if follow.Status != 204 {
		t.Fatal(string(follow.Body))
	}
	a.waitFor(t, id, "done")
	a.gateway.mu.Lock()
	defer a.gateway.mu.Unlock()
	if len(a.gateway.requests) != 2 || a.gateway.requests[0]["model"] != "tier-first" ||
		a.gateway.requests[1]["model"] != "tier-second" {
		t.Fatalf("actual models: %+v", a.gateway.requests)
	}
	mustContain(t, do(t, a.env.app, "GET", "/api/tasks/"+id, "", headers).Body, `"model":"tier-second"`, `"effort":"high"`)
	events := do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", headers)
	mustContain(t, events.Body, "Orchestrator now uses tier-second")
}

func saveThinking(t *testing.T, a agentRig, prefs map[string]any, provider, model string) {
	t.Helper()
	prefs["thinking"] = map[string]string{"providerId": provider, "model": model, "effort": "high"}
	body, _ := json.Marshal(prefs)
	if res := do(t, a.env.app, "PUT", "/api/preferences", string(body), jsonCookie(a.cookie)); res.Status != 200 {
		t.Fatal(string(res.Body))
	}
}

func TestLongMessagesBecomeCompleteProjectDocumentsOnCreateAndFollowUp(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t, finishCall("Read the document"), finishCall("Read the next document"))
	text := "  Start\n" + strings.Repeat("猫🙂resume ", 15_000) + "\nEND-OF-USER-REQUEST  "
	body, _ := json.Marshal(map[string]any{"projectId": a.project, "prompt": text, "team": false})
	res := do(t, a.env.app, "POST", "/api/tasks", string(body), jsonCookie(a.cookie))
	if res.Status != 201 {
		t.Fatalf("long create: %d %s", res.Status, res.Body)
	}
	id, _ := res.JSON(t)["id"].(string)
	a.waitFor(t, id, "done")
	checkOriginalInput(t, a, id, text)
	events := do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(a.cookie))
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
	res = do(t, a.env.app, "POST", "/api/tasks/"+id+"/messages", string(body), jsonCookie(a.cookie))
	if res.Status != 204 {
		t.Fatalf("long follow-up: %d %s", res.Status, res.Body)
	}
	a.waitFor(t, id, "done")
	checkOriginalInput(t, a, id, text+"FOLLOW-UP-END")
}

func checkOriginalInput(t *testing.T, a agentRig, id, original string) {
	t.Helper()
	var docs []struct {
		Path string `json:"path"`
	}
	res := do(t, a.env.app, "GET", "/api/tasks/"+id+"/documents", "", jsonCookie(a.cookie))
	if err := json.Unmarshal(res.Body, &docs); err != nil || res.Status != 200 {
		t.Fatalf("documents: %d %s", res.Status, res.Body)
	}
	for _, doc := range docs {
		isOriginal := strings.HasSuffix(doc.Path, ".txt") && !strings.Contains(doc.Path, "/part-")
		if !strings.Contains(doc.Path, "/input/") || !isOriginal {
			continue
		}
		file := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/file/raw?path="+doc.Path, "", jsonCookie(a.cookie))
		if file.Status == 200 && string(file.Body) == original {
			checkInputParts(t, a, strings.TrimSuffix(doc.Path, ".txt"), original)
			return
		}
	}
	t.Fatal("the exact original long message was not available through the project's file API")
}

func checkInputParts(t *testing.T, a agentRig, base, original string) {
	t.Helper()
	var combined strings.Builder
	for part := 1; ; part++ {
		name := fmt.Sprintf("%s/part-%04d.txt", base, part)
		file := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/file?path="+name, "", jsonCookie(a.cookie))
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
