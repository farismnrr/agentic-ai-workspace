package projects_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestTaskPinEndpointsPersistAndReturnPinnedAt(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	id := a.Start(t, "Pin this")
	a.WaitFor(t, id, "done")
	path := "/api/tasks/" + id + "/pin"

	pinned := apptest.Do(t, a.Environment.App, "PUT", path, "", apptest.JSONCookie(a.Cookie))
	requireStatus(t, pinned, 200, "pin")
	var response struct {
		PinnedAt *time.Time `json:"pinnedAt"`
	}
	if err := json.Unmarshal(pinned.Body, &response); err != nil || response.PinnedAt == nil {
		t.Fatalf("pin response should include pinnedAt: %s (%v)", pinned.Body, err)
	}
	assertRepeatedPinKeepsTime(t, a, id, path)
	assertUnpinLifecycle(t, a, path)
}

func assertRepeatedPinKeepsTime(t *testing.T, a apptest.AgentRig, id, path string) {
	t.Helper()
	const original = int64(12345)
	const query = `UPDATE tasks SET pinned_at = ? WHERE id = ?`
	if _, err := a.Environment.DB.ExecContext(context.Background(), query, original, id); err != nil {
		t.Fatal(err)
	}
	repeated := apptest.Do(t, a.Environment.App, "PUT", path, "", apptest.JSONCookie(a.Cookie))
	requireStatus(t, repeated, 200, "repeat pin")
	var response struct {
		PinnedAt *time.Time `json:"pinnedAt"`
	}
	if err := json.Unmarshal(repeated.Body, &response); err != nil {
		t.Fatalf("decode repeated pin: %v", err)
	}
	if response.PinnedAt == nil || response.PinnedAt.Unix() != original {
		t.Fatalf("repeat pin must preserve its original time: %s", repeated.Body)
	}

	listed := apptest.Do(t, a.Environment.App, "GET", "/api/tasks", "", apptest.JSONCookie(a.Cookie))
	if listed.Status != 200 || !strings.Contains(string(listed.Body), `"pinnedAt":"1970-01-01T03:25:45Z"`) {
		t.Fatalf("task list should include pinnedAt: %d %s", listed.Status, listed.Body)
	}
}

func assertUnpinLifecycle(t *testing.T, a apptest.AgentRig, path string) {
	t.Helper()
	unpinned := apptest.Do(t, a.Environment.App, "DELETE", path, "", apptest.JSONCookie(a.Cookie))
	if unpinned.Status != 200 || !strings.Contains(string(unpinned.Body), `"pinnedAt":null`) {
		t.Fatalf("unpin should return a null pinnedAt: %d %s", unpinned.Status, unpinned.Body)
	}
	repeatedAgain := apptest.Do(t, a.Environment.App, "DELETE", path, "", apptest.JSONCookie(a.Cookie))
	requireStatus(t, repeatedAgain, 200, "repeat unpin")
}

func TestPinEndpointsHideUnknownAndOtherUsersTasks(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	id := a.Start(t, "Private task")
	a.WaitFor(t, id, "done")
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"pin-eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	assertHiddenPin(t, a, other, "PUT", id)
	assertHiddenPin(t, a, other, "PUT", "unknown-task")
	assertHiddenPin(t, a, other, "DELETE", id)
	assertHiddenPin(t, a, other, "DELETE", "unknown-task")
}

func assertHiddenPin(t *testing.T, a apptest.AgentRig, cookie, method, id string) {
	t.Helper()
	got := apptest.Do(t, a.Environment.App, method, "/api/tasks/"+id+"/pin", "", apptest.JSONCookie(cookie))
	requireStatus(t, got, 404, method+" pin for "+id)
}
