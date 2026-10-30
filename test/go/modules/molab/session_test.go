package molab_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	molab "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
)

func TestPairingIsVolatileAndTaskScoped(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	sessions := molab.NewSessions(func() time.Time { return now })
	state, err := sessions.Put("task-1", "alice", "project-1", "https://sb-f59954d0007694b7.sb.molab.run/", "synthetic-secret-123")
	if err != nil || !state.Connected {
		t.Fatalf("pair: %v, %#v", err, state)
	}
	if _, err = sessions.Get("task-1", "bob", "project-1"); !errors.Is(err, molab.ErrUnavailable) {
		t.Fatal("cross-user token exposed")
	}
	if _, err = sessions.Get("task-2", "alice", "project-1"); !errors.Is(err, molab.ErrUnavailable) {
		t.Fatal("cross-task token exposed")
	}
	if _, err = sessions.Get("task-1", "alice", "project-2"); !errors.Is(err, molab.ErrUnavailable) {
		t.Fatal("cross-project token exposed")
	}
	data, err := json.Marshal(sessions.Status("task-1", "alice", "project-1"))
	if err != nil || strings.Contains(string(data), "synthetic-secret") {
		t.Fatal("status leaked token")
	}
	v, err := sessions.Get("task-1", "alice", "project-1")
	if err != nil || v.Token != "synthetic-secret-123" {
		t.Fatal("correct task cannot use token")
	}
	previous := v
	_, err = sessions.Put("task-1", "alice", "project-1", "https://sb-f59954d0007694b7.sb.molab.run/", "synthetic-secret-123")
	if err != nil {
		t.Fatal(err)
	}
	sessions.MarkInspected("task-1", previous)
	v, _ = sessions.Get("task-1", "alice", "project-1")
	if v.Inspected {
		t.Fatal("old pairing with reused token changed inspection state")
	}
	sessions.MarkInspected("task-1", v)
	v, _ = sessions.Get("task-1", "alice", "project-1")
	if !v.Inspected {
		t.Fatal("inspection state missing")
	}
	sessions.Forget("task-1")
	if _, err = sessions.Get("task-1", "alice", "project-1"); !errors.Is(err, molab.ErrUnavailable) {
		t.Fatal("disconnect failed")
	}
}

func TestPairingExpiresAndRestarts(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	sessions := molab.NewSessions(func() time.Time { return now })
	_, err := sessions.Put("task-1", "alice", "p", "https://sb-a.sb.molab.run/", "synthetic-secret-123")
	if err != nil {
		t.Fatal(err)
	}
	if molab.NewSessions(func() time.Time { return now }).Status("task-1", "alice", "p").Connected {
		t.Fatal("restart restored secret")
	}
	now = now.Add(molab.SessionTTL)
	if _, err = sessions.Get("task-1", "alice", "p"); !errors.Is(err, molab.ErrUnavailable) {
		t.Fatal("expired secret is readable")
	}
}

func TestOnlyExactSecureMoLabOrigins(t *testing.T) {
	invalid := []string{
		"http://sb-a.sb.molab.run/", "https://localhost/", "https://127.0.0.1/",
		"https://sb-a.sb.molab.run.evil.com/", "https://sb-a.sb.molab.run@evil.com/",
		"https://evil.com@sb-a.sb.molab.run/", "https://sb-a.sb.molab.run:444/",
		"https://sb-a.sb.molab.run/path", "https://sb-a.sb.molab.run/?x=1",
		"file:///etc/passwd", "https://sb-a.sb.molab.run/#frag",
	}
	for _, u := range invalid {
		if _, err := molab.Validate(u, "synthetic-secret-123"); err == nil {
			t.Errorf("unsafe URL accepted: %s", u)
		}
	}
	if _, err := molab.Validate("https://sb-a.sb.molab.run/", "\nEXFIL"); err == nil {
		t.Error("newline token accepted")
	}
}

func TestDisconnectAndReplacementCancelInFlightExecution(t *testing.T) {
	sessions := molab.NewSessions(time.Now)
	_, err := sessions.Put("task", "alice", "p", "https://sb-test.sb.molab.run/", "synthetic-secret-123")
	if err != nil {
		t.Fatal(err)
	}
	original, _ := sessions.Get("task", "alice", "p")
	running, stop := original.Bound(context.Background())
	defer stop()
	_, err = sessions.Put("task", "alice", "p", "https://sb-test.sb.molab.run/", "synthetic-secret-456")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-running.Done():
	case <-time.After(time.Second):
		t.Fatal("replacing pairing did not cancel old execution")
	}
	replacement, _ := sessions.Get("task", "alice", "p")
	pending, done := replacement.Bound(context.Background())
	defer done()
	sessions.Forget("task")
	select {
	case <-pending.Done():
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel execution")
	}
}
