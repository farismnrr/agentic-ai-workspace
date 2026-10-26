package runner_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

type closedTasks struct{}

func (closedTasks) TaskLive(_ context.Context, _, task string) (bool, error) {
	return task != "over", nil
}

// grantEngineWith is the toolbox engine with task sockets on, size places, and the given idea of which tasks are open.
func grantEngineWith(
	t *testing.T, size int, scopes runner.ScopeCheck,
) (*runner.Engine, *egressproxy.Grants, *egressproxy.Hub) {
	t.Helper()

	if toolboxImage() == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	short, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}

	grants := egressproxy.NewGrants(egressproxy.DefaultLimits(), nil)
	hub := egressproxy.NewHub(short, egressproxy.New([]string{"pypi.org"})).WithGrants(grants)

	t.Cleanup(func() { grants.Close(); hub.Close(); _ = os.RemoveAll(short) })

	return runner.NewEngine(runner.EngineConfig{
		Size:   size,
		Extras: runner.Extras{CacheDir: t.TempDir(), Proxy: hub, Scopes: scopes},
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			return sandbox.Open(ctx, sandbox.Config{
				Image: toolboxImage(), Limits: sandbox.DefaultLimits(), UID: os.Getuid(), GID: os.Getgid(),
			})
		},
	}), grants, hub
}

// A vendor's program works for one task. Its container gets that task's socket: a grant of that task reaches the
// proxy's authorization, the same grant seen from another task's container is refused.
func TestAVendorProgramContainerGetsOnlyItsOwnTasksSocket(t *testing.T) {
	eng, grants, _ := grantEngineWith(t, 2, closedTasks{})
	host := "granted-for-b.example.com"

	scope := egressproxy.Scope{Project: "p1", Task: "b"}
	if err := grants.Allow("g1", scope, host, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	probe := func(task string) string {
		ctx := runner.WithTask(context.Background(), "p1", task)

		docker, spec, release, err := eng.Lease(ctx, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		defer release()

		spec.Env = []string{"HTTPS_PROXY=http://127.0.0.1:3128", "PATH=" + sandbox.Path}
		spec.Argv = []string{"sh", "-c", "curl -sS -m 20 -o /dev/null https://" + host + " 2>&1; ls -A /run/egress"}

		var out bytes.Buffer
		if _, err = docker.Run(ctx, spec, &out); err != nil {
			t.Fatal(err)
		}

		return out.String()
	}

	if out := probe("a"); !strings.Contains(out, "403") || !strings.HasSuffix(strings.TrimSpace(out), "p.sock") {
		t.Errorf("task a: %q", out)
	}

	if out := probe("b"); !strings.Contains(out, "502") || !strings.HasSuffix(strings.TrimSpace(out), "p.sock") {
		t.Errorf("task b: %q", out)
	}
}

func TestAVendorProgramForATaskThatIsOverDoesNotStartAndFreesItsPlace(t *testing.T) {
	eng, _, _ := grantEngineWith(t, 1, closedTasks{})

	for range 3 {
		_, _, _, err := eng.Lease(runner.WithTask(context.Background(), "p1", "over"), t.TempDir())
		if !errors.Is(err, runner.ErrTaskNotOpen) {
			t.Fatalf("got %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, _, release, err := eng.Lease(runner.WithTask(ctx, "p1", "open"), t.TempDir())
	if err != nil {
		t.Fatalf("the place must have been given back: %v", err)
	}

	release()
}

func TestWithoutAScopeAndForSitesTheWorkspaceSocketIsUsedAndCarriesNoGrants(t *testing.T) {
	eng, _, hub := grantEngineWith(t, 2, closedTasks{})
	dir := t.TempDir()

	want, err := hub.Dir(context.Background(), filepath.Base(dir))
	if err != nil {
		t.Fatal(err)
	}

	_, spec, release, err := eng.Lease(context.Background(), dir)
	if err != nil || spec.Egress != want {
		t.Fatalf("no scope: %q %v, want %q", spec.Egress, err, want)
	}

	release()

	docker, err := eng.Docker(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	site := eng.SiteSpec(runner.WithTask(context.Background(), "p1", "b"), dir, docker, []string{"true"}, nil)
	if site.Egress != want {
		t.Fatalf("a site must never get a task socket: %q, want %q", site.Egress, want)
	}
}
