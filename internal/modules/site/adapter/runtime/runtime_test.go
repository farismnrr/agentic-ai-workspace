package runtime_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/runtime"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

type folders struct{ dir string }

func (f folders) Dir(string) (string, error) { return f.dir, nil }

// newRuntime needs the toolbox image (SANDBOX_TEST_IMAGE, built by `make sandbox-image`).
func newRuntime(t *testing.T) *runtime.Runtime {
	t.Helper()
	return newRuntimeWithProject(t, t.TempDir())
}

func newRuntimeWithProject(t *testing.T, projectDir string) *runtime.Runtime {
	t.Helper()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	short, err := os.MkdirTemp("", "s")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(short) })

	engine := codingrunner.NewEngine(codingrunner.EngineConfig{
		Size: 2,
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			return sandbox.Open(ctx, sandbox.Config{
				Image: image, Limits: sandbox.DefaultLimits(), UID: os.Getuid(), GID: os.Getgid(),
			})
		},
	})

	return runtime.New(engine, folders{dir: projectDir}, short)
}

func TestAnAppInTheSandboxIsReachableOnlyThroughItsSocket(t *testing.T) {
	rt := newRuntime(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	defer cancel()

	id, err := rt.Start(ctx, application.StartSpec{
		ProjectID: "p1", BasePath: "/run/demo/abc", Port: 8000,
		Command: `echo "base=$BASE_PATH" > index.html && exec python3 -m http.server "$PORT" --bind 127.0.0.1`,
	})
	if err != nil {
		t.Fatal(err)
	}

	defer rt.Stop(ctx, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://site/index.html", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := runtime.Client(rt.Socket("p1"), 10*time.Second).Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if !strings.Contains(string(body), "base=/run/demo/abc") {
		t.Fatalf("the app must see its base path, got %q", body)
	}

	if !rt.Alive(ctx, id) {
		t.Fatal("the container should be running")
	}

	rt.Stop(ctx, id)

	if rt.Alive(ctx, id) {
		t.Fatal("a stopped site must not be alive")
	}
}

func TestAnAppThatDiesIsReportedWithItsOutput(t *testing.T) {
	rt := newRuntime(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	defer cancel()

	_, err := rt.Start(ctx, application.StartSpec{
		ProjectID: "p2", BasePath: "/run/x/y", Port: 8001, Command: `echo "boom: no module named flask" >&2; exit 3`,
	})
	if !errors.Is(err, domain.ErrNotStarted) || !strings.Contains(err.Error(), "no module named flask") {
		t.Fatalf("the model needs the app's own output to fix it, got %v", err)
	}
}
