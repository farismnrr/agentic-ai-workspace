package runner_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

type openTasks struct{}

func (openTasks) TaskLive(context.Context, string, string) (bool, error) { return true, nil }

type noEnv struct{}

func (noEnv) Values(context.Context, string) (map[string]string, error) { return nil, nil }

// grantEngine is the toolbox engine with task sockets on and every task live.
func grantEngine(t *testing.T) (*runner.Engine, *egressproxy.Grants) {
	t.Helper()

	image := toolboxImage()
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	short, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(short) })

	grants := egressproxy.NewGrants(egressproxy.DefaultLimits(), nil)
	hub := egressproxy.NewHub(short, egressproxy.New([]string{"pypi.org"})).WithGrants(grants)

	t.Cleanup(func() { grants.Close(); hub.Close() })

	return runner.NewEngine(runner.EngineConfig{
		Size:   2,
		Extras: runner.Extras{CacheDir: t.TempDir(), Proxy: hub, Scopes: openTasks{}},
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			return sandbox.Open(ctx, sandbox.Config{
				Image: image, Limits: sandbox.DefaultLimits(), UID: os.Getuid(), GID: os.Getgid(),
			})
		},
	}), grants
}

func runAs(t *testing.T, eng *runner.Engine, dir, task, line string) string {
	t.Helper()

	cmd, err := domain.CommandTask(line, 60)
	if err != nil {
		t.Fatal(err)
	}

	r := runner.New(dir, eng).WithProjectEnv("p1", noEnv{})

	res, err := r.Run(runner.WithTask(context.Background(), "p1", task), cmd)
	if err != nil {
		t.Fatal(err)
	}

	return res.Output
}

// The grant is for task b only. From a real container, task a's socket answers 403 (not authorized) while task b's
// is authorized and fails later, when the made-up name does not resolve (502): the two sandboxes get different
// answers from the same host name.
func TestAGrantIsUsableOnlyFromItsOwnTasksContainer(t *testing.T) {
	eng, grants := grantEngine(t)
	dir := t.TempDir()
	host := "granted-for-b.example.com"

	scope := egressproxy.Scope{Project: "p1", Task: "b"}
	if err := grants.Allow("g1", scope, host, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	probe := "curl -sS -m 20 -o /dev/null https://" + host + " 2>&1; true"

	if out := runAs(t, eng, dir, "a", probe); !strings.Contains(out, "403") {
		t.Errorf("task a must be refused (403): %q", out)
	}

	if out := runAs(t, eng, dir, "b", probe); !strings.Contains(out, "502") {
		t.Errorf("task b must be let through the proxy's authorization (502, the name does not resolve): %q", out)
	}
}

func TestTheContainerSeesOnlyItsOwnSocketAndCannotChangeIt(t *testing.T) {
	eng, _ := grantEngine(t)
	dir := t.TempDir()
	script := `ls -A /run/egress; echo ---;
find / -path /proc -prune -o \( -name 'p.sock' -o -name '.grants.lock' \) -print 2>/dev/null; echo ---;
touch /run/egress/x 2>&1; rm -f /run/egress/p.sock 2>&1; ls /run/egress`

	for _, task := range []string{"a", "b"} {
		out := runAs(t, eng, dir, task, script)
		parts := strings.Split(out, "---")

		if len(parts) != 3 || strings.TrimSpace(parts[0]) != "p.sock" {
			t.Fatalf("task %s: /run/egress must hold only its socket: %q", task, out)
		}

		if found := strings.Fields(parts[1]); len(found) != 1 || found[0] != "/run/egress/p.sock" {
			t.Errorf("task %s: sockets visible in the container: %v", task, found)
		}

		if !strings.Contains(parts[2], "Read-only") || !strings.HasSuffix(strings.TrimSpace(parts[2]), "p.sock") {
			t.Errorf("task %s: the socket folder must be read-only: %q", task, parts[2])
		}
	}
}

func toolboxImage() string { return os.Getenv("SANDBOX_TEST_IMAGE") }
