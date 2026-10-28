package stream_test

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestAReturningViewerOnlyGetsWhatItMissed(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	base := serve(t, a.Environment.App)
	id := a.Start(t, "Quick one")
	a.WaitFor(t, id, "done")
	events := a.Thread(t, id)
	last := events[len(events)-1]

	url := base + "/api/tasks/" + id + "/stream"
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	request.Header.Set("Cookie", a.Cookie)
	request.Header.Set("Last-Event-ID", strconv.FormatInt(last.Version, 10))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()

	reader := bufio.NewReader(response.Body)
	_, _ = reader.ReadString('\n')
	_, _ = reader.ReadString('\n')
	_, _ = reader.ReadString('\n')
	line, _ := reader.ReadString('\n')
	if strings.Contains(line, "receipt") || !strings.Contains(line, "update") {
		t.Fatalf("a viewer that has everything should get no repeated entries: %q", line)
	}
}

func TestAReconnectReplaysAnUpdatedOlderEvent(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	base := serve(t, a.Environment.App)
	id := a.Start(t, "Quick one")
	a.WaitFor(t, id, "done")
	events := a.Thread(t, id)
	lastVersion := events[len(events)-1].Version
	setHistoricalEventText(t, a.Environment.DB, id)
	assertChangedEventAPI(t, a, id, lastVersion)
	assertChangedEventStream(t, a.Cookie, base, id, lastVersion)
}

func setHistoricalEventText(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	var body string
	if err := db.QueryRowContext(context.Background(),
		`SELECT body FROM task_events WHERE task_id = ? AND seq = 1`, id).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var changed map[string]any
	if err := json.Unmarshal([]byte(body), &changed); err != nil {
		t.Fatal(err)
	}
	changed["text"] = "Updated historical event"
	encoded, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, updateErr := db.ExecContext(context.Background(), `UPDATE task_events SET body = ?, version =
		(SELECT MAX(version) + 1 FROM task_events WHERE task_id = ?) WHERE task_id = ? AND seq = 1`,
		string(encoded), id, id); updateErr != nil {
		t.Fatal(updateErr)
	}
}

func assertChangedEventAPI(t *testing.T, a apptest.AgentRig, id string, version int64) {
	t.Helper()
	listed := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events?after="+strconv.FormatInt(version, 10),
		"", apptest.JSONCookie(a.Cookie))
	if !strings.Contains(string(listed.Body), "Updated historical event") {
		t.Fatalf("changed event was not returned after its prior cursor: %s", listed.Body)
	}
}

func assertChangedEventStream(t *testing.T, cookie, base, id string, version int64) {
	t.Helper()
	url := base + "/api/tasks/" + id + "/stream"
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	request.Header.Set("Cookie", cookie)
	request.Header.Set("Last-Event-ID", strconv.FormatInt(version, 10))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	for range 8 {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
		if strings.Contains(line, "Updated historical event") {
			return
		}
	}
	t.Fatal("reconnected stream did not replay the changed historical event")
}
