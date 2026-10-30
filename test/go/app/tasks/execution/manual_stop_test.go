package execution_test

import (
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestManualStopRunningAndQueuedInternalTasksStayDone(t *testing.T) {
	rig := apptest.NewAgentRig(t, apptest.Plan(), apptest.FinishCall("Unexpected completion"))
	hold := make(chan struct{})
	rig.Gateway.Hold = hold
	defer func() {
		select {
		case <-hold:
		default:
			close(hold)
		}
	}()
	running := rig.Start(t, "Hold this task while running")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rig.Gateway.Mu.Lock()
		n := len(rig.Gateway.Requests)
		rig.Gateway.Mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	rig.Gateway.Mu.Lock()
	n := len(rig.Gateway.Requests)
	rig.Gateway.Mu.Unlock()
	if n == 0 {
		t.Fatal("model never started")
	}
	queued := rig.Start(t, "Queued behind first task")
	if got := rig.Status(t, queued); got != "queued" {
		t.Fatalf("not queued: %s", got)
	}
	for _, id := range []string{queued, running} {
		resp := apptest.Do(t, rig.Environment.App, http.MethodPost, "/api/tasks/"+id+"/cancel", `{}`, apptest.JSONCookie(rig.Cookie))
		if resp.Status != http.StatusNoContent {
			t.Fatalf("cancel %s: %d %s", id, resp.Status, resp.Body)
		}
		if got := rig.Status(t, id); got != "done" {
			t.Fatalf("manual stop %s got %s, expected done", id, got)
		}
	}
	close(hold)
	time.Sleep(300 * time.Millisecond)
	for _, id := range []string{queued, running} {
		if got := rig.Status(t, id); got != "done" {
			t.Fatalf("late worker settle re-opened %s as %s", id, got)
		}
		events := apptest.Do(t, rig.Environment.App, http.MethodGet, "/api/tasks/"+id+"/events", "", apptest.JSONCookie(rig.Cookie))
		if !strings.Contains(string(events.Body), "Stopped by you") {
			t.Fatalf("manual stop note missing: %s", events.Body)
		}
	}
}

func TestRealModelFailureStillNeedsAttention(t *testing.T) {
	rig := apptest.NewAgentRig(t) // No scripted model response: this is a real model error.
	id := rig.Start(t, "Make a request that fails")
	rig.WaitFor(t, id, "failed")
	stop := apptest.Do(t, rig.Environment.App, http.MethodPost, "/api/tasks/"+id+"/cancel", `{}`, apptest.JSONCookie(rig.Cookie))
	if stop.Status != http.StatusConflict || rig.Status(t, id) != "failed" {
		t.Fatalf("genuine model failure should stay failed, cancel=%d status=%s", stop.Status, rig.Status(t, id))
	}
}
