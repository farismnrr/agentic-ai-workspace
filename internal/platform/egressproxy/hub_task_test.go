package egressproxy

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskSocketsAreOffUnlessTheHubHasGrants(t *testing.T) {
	hub := NewHub(t.TempDir(), New(nil))
	t.Cleanup(hub.Close)

	if _, err := hub.TaskDir(context.Background(), "ws", scopeA); !errors.Is(err, ErrGrantsOff) {
		t.Fatalf("got %v", err)
	}

	hub.WithGrants(NewGrants(DefaultLimits(), nil))

	for _, scope := range []Scope{{}, {Project: "p"}, {Task: "t"}} {
		if _, err := hub.TaskDir(context.Background(), "ws", scope); !errors.Is(err, ErrBadGrant) {
			t.Errorf("%+v: %v", scope, err)
		}
	}

	if _, err := hub.TaskDir(context.Background(), "", scopeA); !errors.Is(err, ErrBadGrant) {
		t.Errorf("no workspace: %v", err)
	}
}

func TestTheOlderPathsAreUnchangedOnATaskSocket(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	dir, err := r.hub.TaskDir(context.Background(), "ws", scopeA, "github.com")
	if err != nil {
		t.Fatal(err)
	}

	for target, want := range map[string]int{
		"pypi.org:443": http.StatusOK, "github.com:443": http.StatusOK, "evil.example.com:443": http.StatusForbidden,
		"pypi.org:80": http.StatusForbidden, "localhost:5432": http.StatusForbidden,
	} {
		if got := r.openAt(dir, target).status; got != want {
			t.Errorf("%s: %d, want %d", target, got, want)
		}
	}

	base := r.hub.proxy
	if _, ok := base.hosts[goodHost]; ok || len(base.targets) != 0 {
		t.Fatal("a grant must never be added to the static list or the exact targets")
	}
}

func TestAWorkspaceSocketNeverServesGrants(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	dir, err := r.hub.Dir(context.Background(), "ws")
	if err != nil {
		t.Fatal(err)
	}

	if c := r.openAt(dir, goodHost+":443"); c.status != http.StatusForbidden {
		t.Fatalf("%d", c.status)
	}
}

func TestRemoveTaskStopsItsSocketAndRevokesItsGrants(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)
	r.allow("g2", scopeB, time.Minute)

	dir, _ := r.hub.TaskDir(context.Background(), "ws", scopeA)
	other, _ := r.hub.TaskDir(context.Background(), "ws", scopeB)
	c := established(t, r, scopeA)

	r.hub.RemoveTask(scopeA)

	if !c.closedWithin(time.Second) {
		t.Fatal("the open tunnel must close")
	}

	if _, err := os.Stat(filepath.Join(dir, socketName)); !os.IsNotExist(err) {
		t.Fatalf("the socket must be gone: %v", err)
	}

	if _, err := r.grants.Lease(scopeA, goodHost); err == nil {
		t.Fatal("the grant must be gone")
	}

	if _, err := os.Stat(filepath.Join(other, socketName)); err != nil {
		t.Fatalf("another task's socket must stay: %v", err)
	}

	r.hub.RemoveTask(Scope{})
}

func TestSocketFilesAreNotWorldAccessible(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	dir, _ := r.hub.TaskDir(context.Background(), "ws", scopeA)

	info, err := os.Stat(filepath.Join(dir, socketName))
	if err != nil || info.Mode().Perm()&0o007 != 0 {
		t.Fatalf("%v %v", info.Mode(), err)
	}
}
