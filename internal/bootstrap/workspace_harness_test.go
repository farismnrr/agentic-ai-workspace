package bootstrap

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox/sandboxtest"
)

var (
	sandboxOnce sync.Once
	sandboxPath string
	errSandbox  error
)

// testSandboxDir is the folder holding the shared toolbox the tests run tasks in, built once.
func testSandboxDir() string {
	sandboxOnce.Do(func() {
		if sandboxPath, errSandbox = sandboxtest.Ensure(); errSandbox == nil {
			errSandbox = sandboxtest.Probe(sandboxPath)
		}
	})

	return sandboxPath
}

// requireToolbox skips a test that runs a sandboxed task when no toolbox could be had.
func requireToolbox(t *testing.T) {
	t.Helper()

	if testSandboxDir(); errSandbox != nil {
		t.Skipf("no sandbox toolbox on this host: %v", errSandbox)
	}
}

// workspaceRoots remembers where each test app keeps its per-person workspaces.
var workspaceRoots sync.Map

var workspaceFixtures = map[string]string{
	"hello.txt":             "hello from the workspace\n",
	".workspace/tasks.json": `{"tasks":{"echo":{"description":"Say hello","command":["echo","hello task"]}}}`,
}

// userWorkspace is the folder of the person behind cookie.
func userWorkspace(t *testing.T, app *fiber.App, cookie string) string {
	t.Helper()

	me := do(t, app, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)
	id, _ := me["id"].(string)
	base, _ := workspaceRoots.Load(app)

	root, _ := base.(string)
	if id == "" || root == "" {
		t.Fatalf("cannot locate the workspace of %v", me)
	}

	return filepath.Join(root, id)
}

// seedWorkspace gives the person behind cookie the sample files the tests read.
func seedWorkspace(t *testing.T, app *fiber.App, cookie string) {
	t.Helper()

	dir := userWorkspace(t, app, cookie)

	for name, body := range workspaceFixtures {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
