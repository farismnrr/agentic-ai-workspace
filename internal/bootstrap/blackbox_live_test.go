package bootstrap

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	webdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

var pictureURL = regexp.MustCompile(`url: (https?://[^\s"\\]+)`)

// TestLiveTeamResearchesThenFindsAPicture drives the whole app over HTTP with a scripted model, but real web
// services: your SearXNG, Openverse, Wikimedia Commons and the open internet. It needs the network, so it only
// runs when BLACKBOX_LIVE is set. BLACKBOX_SEARXNG says where SearXNG is (default http://127.0.0.1:8888).
//
//	BLACKBOX_LIVE=1 go test ./internal/bootstrap -run Live
func TestLiveTeamResearchesThenFindsAPicture(t *testing.T) {
	if os.Getenv("BLACKBOX_LIVE") == "" {
		t.Skip("set BLACKBOX_LIVE=1 to run against the real web")
	}

	searx := os.Getenv("BLACKBOX_SEARXNG")
	if searx == "" {
		searx = "http://127.0.0.1:8888"
	}

	env := newEnvWith(t, func(c *config.Config) { c.SearxngURL = searx })
	a := rigOn(t, env,
		toolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Researcher: look up coffee shop sites", "state": "pending"},
		}}),
		toolCall("d1", "delegate", map[string]string{"role": "planner", "brief": "Plan a coffee shop site"}),
		toolCall("d2", "delegate", map[string]string{"role": "researcher", "brief": "Look up coffee shop sites"}),
		toolCall("s1", "web_search", map[string]string{"query": "warung kopi bandung landing page"}),
		toolCall("s2", "fetch_page", map[string]string{"url": "http://127.0.0.1:8081/api/health"}),
		toolCall("s3", "fetch_page", map[string]string{"url": "https://en.wikipedia.org/wiki/Bandung"}),
		toolCall("s4", "report", map[string]string{"summary": "Notes", "document": "## Discovery\n- Bandung coffee shops"}),
		toolCall("d3", "delegate", map[string]string{"role": "media", "brief": "Find a coffee picture"}),
		toolCall("m1", "search_images", map[string]string{"query": "coffee cup"}),
		"",
		toolCall("m3", "report", map[string]string{"summary": "Saved a picture"}),
		finishCall("Done"),
	)
	a.gateway.late = map[int]func([]map[string]any) string{9: func(requests []map[string]any) string {
		raw, _ := json.Marshal(requests[len(requests)-1])
		found := pictureURL.FindSubmatch(raw)

		if found == nil {
			return toolCall("m2", "report", map[string]string{"summary": "no pictures were offered"})
		}

		return toolCall("m2", "download_asset", map[string]string{"url": string(found[1]), "path": "hero"})
	}}

	id := a.startTeam(t, "Buatkan website untuk warung kopi di Bandung")
	finishLive(t, a, id)

	checkLive(t, a, id, a.toolResults(t))
}

func checkLive(t *testing.T, a agentRig, id string, requests []string) {
	t.Helper()

	if len(requests) < 11 {
		t.Fatalf("the script did not run to the end: %d model requests", len(requests))
	}

	checkResearch(t, a, id, requests)
	checkPicture(t, a, requests)
}

// checkResearch covers the Researcher's part: held-back Planner, marked results, refused internal address.
func checkResearch(t *testing.T, a agentRig, id string, requests []string) {
	t.Helper()

	if !strings.Contains(requests[2], "hand the Researcher a job") {
		t.Errorf("the Planner was not held back until the Researcher ran")
	}

	if !strings.Contains(requests[4], "written by strangers") {
		t.Errorf("search results are not marked as coming from strangers")
	}

	if !strings.Contains(requests[5], webdomain.ErrBlocked.Error()) {
		t.Errorf("an internal address was not refused")
	}

	if !strings.Contains(requests[6], "Bandung") {
		t.Errorf("a real page was not read")
	}

	task := do(t, a.env.app, "GET", "/api/tasks/"+id, "", jsonCookie(a.cookie)).JSON(t)
	if research, _ := task["research"].(string); !strings.Contains(research, "Bandung coffee shops") {
		t.Errorf("the research notes were not saved on the task: %v", task["research"])
	}
}

// checkPicture covers Media's part: a real licensed picture saved under assets/ with its credit line.
func checkPicture(t *testing.T, a agentRig, requests []string) {
	t.Helper()

	data, err := os.ReadFile(a.file("assets/hero.jpg"))
	if err != nil {
		data, err = os.ReadFile(a.file("assets/hero.png"))
	}

	if err != nil || len(data) < 1000 {
		t.Errorf("no real picture was saved under assets/: %v (%d bytes)", err, len(data))
	}

	if !strings.Contains(requests[10], "CREDITS.md") {
		t.Errorf("the credit line did not come back")
	}
}
