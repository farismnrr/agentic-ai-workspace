package bootstrap

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTaskPinEndpointsPersistAndReturnPinnedAt(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	id := a.start(t, "Pin this")
	a.waitFor(t, id, "done")
	path := "/api/tasks/" + id + "/pin"

	pinned := do(t, a.env.app, "PUT", path, "", jsonCookie(a.cookie))
	if pinned.Status != 200 {
		t.Fatalf("pin = %d %s", pinned.Status, pinned.Body)
	}
	var response struct {
		PinnedAt *time.Time `json:"pinnedAt"`
	}
	if err := json.Unmarshal(pinned.Body, &response); err != nil || response.PinnedAt == nil {
		t.Fatalf("pin response should include pinnedAt: %s (%v)", pinned.Body, err)
	}
	assertRepeatedPinKeepsTime(t, a, id, path)
	assertUnpinLifecycle(t, a, path)
}

func assertRepeatedPinKeepsTime(t *testing.T, a agentRig, id, path string) {
	t.Helper()
	value, _ := testStores.Load(a.env.app)
	store := value.(testStore) //nolint:forcetypeassert // test store is registered by newEnvServing
	const original = int64(12345)
	const query = `UPDATE tasks SET pinned_at = ? WHERE id = ?`
	if _, err := store.db.ExecContext(context.Background(), query, original, id); err != nil {
		t.Fatal(err)
	}
	repeated := do(t, a.env.app, "PUT", path, "", jsonCookie(a.cookie))
	if repeated.Status != 200 {
		t.Fatalf("repeat pin = %d %s", repeated.Status, repeated.Body)
	}
	var response struct {
		PinnedAt *time.Time `json:"pinnedAt"`
	}
	if err := json.Unmarshal(repeated.Body, &response); err != nil {
		t.Fatalf("decode repeated pin: %v", err)
	}
	if response.PinnedAt == nil || response.PinnedAt.Unix() != original {
		t.Fatalf("repeat pin must preserve its original time: %s", repeated.Body)
	}

	listed := do(t, a.env.app, "GET", "/api/tasks", "", jsonCookie(a.cookie))
	if listed.Status != 200 || !strings.Contains(string(listed.Body), `"pinnedAt":"1970-01-01T03:25:45Z"`) {
		t.Fatalf("task list should include pinnedAt: %d %s", listed.Status, listed.Body)
	}
}

func assertUnpinLifecycle(t *testing.T, a agentRig, path string) {
	t.Helper()
	unpinned := do(t, a.env.app, "DELETE", path, "", jsonCookie(a.cookie))
	if unpinned.Status != 200 || !strings.Contains(string(unpinned.Body), `"pinnedAt":null`) {
		t.Fatalf("unpin should return a null pinnedAt: %d %s", unpinned.Status, unpinned.Body)
	}
	if repeatedAgain := do(t, a.env.app, "DELETE", path, "", jsonCookie(a.cookie)); repeatedAgain.Status != 200 {
		t.Fatalf("repeat unpin should succeed, got %d %s", repeatedAgain.Status, repeatedAgain.Body)
	}
}

func TestPinEndpointsHideUnknownAndOtherUsersTasks(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	id := a.start(t, "Private task")
	a.waitFor(t, id, "done")
	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"pin-eve@example.com","password":"correct horse"}`, jsonHeaders()))

	assertHiddenPin(t, a, other, "PUT", id)
	assertHiddenPin(t, a, other, "PUT", "unknown-task")
	assertHiddenPin(t, a, other, "DELETE", id)
	assertHiddenPin(t, a, other, "DELETE", "unknown-task")
}

func assertHiddenPin(t *testing.T, a agentRig, cookie, method, id string) {
	t.Helper()
	got := do(t, a.env.app, method, "/api/tasks/"+id+"/pin", "", jsonCookie(cookie)).Status
	if got != 404 {
		t.Errorf("%s pin for %s returned %d, want 404", method, id, got)
	}
}
