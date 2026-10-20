package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// toolboxEngine runs commands in the multi-language image named by SANDBOX_TEST_IMAGE
// (build it with `make sandbox-image`) with a cache and the registry-only proxy.
func toolboxEngine(t *testing.T) *runner.Engine {
	t.Helper()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	short, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(short) })

	hub := egressproxy.NewHub(short, egressproxy.New([]string{
		"pypi.org", "files.pythonhosted.org", "registry.npmjs.org", "proxy.golang.org", "sum.golang.org",
		"storage.googleapis.com",
	}))
	t.Cleanup(hub.Close)

	return runner.NewEngine(runner.EngineConfig{
		Size:   2,
		Extras: runner.Extras{CacheDir: t.TempDir(), Proxy: hub},
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			return sandbox.Open(ctx, sandbox.Config{
				Image: image, Limits: sandbox.DefaultLimits(), UID: os.Getuid(), GID: os.Getgid(),
			})
		},
	})
}

func runCommand(t *testing.T, eng *runner.Engine, dir, line string) string {
	t.Helper()

	task, err := domain.CommandTask(line, 120)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.New(dir, eng).Run(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}

	return result.Output
}

func TestToolboxHasEveryLanguage(t *testing.T) {
	eng := toolboxEngine(t)
	versions := "python3 -V; node -v; go version; rustc -V; java -version 2>&1 | grep -m1 openjdk; bash --version|head -1"
	out := runCommand(t, eng, t.TempDir(), versions)

	for _, want := range []string{"Python 3.13", "v24.", "go1.", "rustc 1.", "openjdk", "GNU bash"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestOnlyRegistriesAreReachableThroughTheProxy(t *testing.T) {
	eng := toolboxEngine(t)
	dir := t.TempDir()

	allowed := runCommand(t, eng, dir, "curl -s -m 30 -o /dev/null -w '%{http_code}' https://pypi.org/")
	if !strings.HasPrefix(allowed, "200") {
		t.Errorf("a registry must be reachable, got %q", allowed)
	}

	for _, line := range []string{
		"curl -sS -m 10 https://example.com 2>&1",
		"curl -sS -m 10 --noproxy '*' https://pypi.org 2>&1",
	} {
		if out := runCommand(t, eng, dir, line); !strings.Contains(out, "curl:") {
			t.Errorf("%s must fail, got %q", line, out)
		}
	}
}

func TestEachWorkspaceHasItsOwnCache(t *testing.T) {
	eng := toolboxEngine(t)
	a, b := filepath.Join(t.TempDir(), "aaaaaaaa"), filepath.Join(t.TempDir(), "bbbbbbbb")

	for _, dir := range []string{a, b} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	runCommand(t, eng, a, "echo mine > /cache/marker")

	if out := runCommand(t, eng, b, "cat /cache/marker 2>&1"); strings.Contains(out, "mine") {
		t.Fatalf("a cache must never be shared between workspaces: %q", out)
	}

	if out := runCommand(t, eng, a, "cat /cache/marker"); !strings.Contains(out, "mine") {
		t.Fatalf("a cache must survive between commands: %q", out)
	}
}

// TestGoModuleRedirectAndCache verifies that a Go module ZIP is downloaded
// through the allowed storage redirect and reused without network access.
func TestGoModuleRedirectAndCache(t *testing.T) {
	eng := toolboxEngine(t)
	dir := t.TempDir()
	first := runCommand(t, eng, dir, "GOPROXY=https://proxy.golang.org go mod download modernc.org/sqlite@v1.59.0 && echo download-ok")
	if !strings.Contains(first, "download-ok") {
		t.Fatalf("Go module download through proxy failed: %s", first)
	}
	second := runCommand(t, eng, dir, "GOPROXY=off go mod download modernc.org/sqlite@v1.59.0 && echo cache-ok")
	if !strings.Contains(second, "cache-ok") {
		t.Fatalf("Go module cache not reused offline: %s", second)
	}
}

func TestPackageManagersInstallThroughTheProxy(t *testing.T) {
	eng := toolboxEngine(t)
	script := `set -e
cd /workspace
pip install --quiet --target pylib six && ls pylib | grep -q six.py && echo pip-ok
npm install --silent --no-audit --no-fund is-odd && test -d node_modules/is-odd && echo npm-ok
go mod init demo >/dev/null 2>&1 && go get rsc.io/quote@v1.5.2 >/dev/null 2>&1 && echo go-ok`

	out := runCommand(t, eng, t.TempDir(), script)
	for _, want := range []string{"pip-ok", "npm-ok", "go-ok"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
