package bootstrap

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestAReturningViewerOnlyGetsWhatItMissed(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	base := a.env.serve(t)
	id := a.start(t, "Quick one")
	a.waitFor(t, id, "done")
	events := a.thread(t, id)
	last := events[len(events)-1]

	url := base + "/api/tasks/" + id + "/stream"
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	request.Header.Set("Cookie", a.cookie)
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

	a := newAgentRig(t, finishCall("Done"))
	base := a.env.serve(t)
	id := a.start(t, "Quick one")
	a.waitFor(t, id, "done")
	events := a.thread(t, id)
	lastVersion := events[len(events)-1].Version
	value, _ := testStores.Load(a.env.app)
	store := value.(testStore) //nolint:forcetypeassert // registered by newEnvServing
	setHistoricalEventText(t, store, id)
	assertChangedEventAPI(t, a, id, lastVersion)
	assertChangedEventStream(t, a.cookie, base, id, lastVersion)
}

func setHistoricalEventText(t *testing.T, store testStore, id string) {
	t.Helper()
	var body string
	if err := store.db.QueryRowContext(context.Background(),
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
	if _, updateErr := store.db.ExecContext(context.Background(), `UPDATE task_events SET body = ?, version =
		(SELECT MAX(version) + 1 FROM task_events WHERE task_id = ?) WHERE task_id = ? AND seq = 1`,
		string(encoded), id, id); updateErr != nil {
		t.Fatal(updateErr)
	}
}

func assertChangedEventAPI(t *testing.T, a agentRig, id string, version int64) {
	t.Helper()
	listed := do(t, a.env.app, "GET", "/api/tasks/"+id+"/events?after="+strconv.FormatInt(version, 10),
		"", jsonCookie(a.cookie))
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
