package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// testImage is small and has the shell tools the tests need.
const testImage = "busybox:1.37"

var (
	dockerOnce sync.Once
	errDocker  error
)

// requireDocker skips a test that runs a task when the host has no usable Docker.
func requireDocker(t *testing.T) {
	t.Helper()

	dockerOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		_, errDocker = sandbox.Open(ctx, sandbox.Config{Image: testImage, Limits: sandbox.DefaultLimits()})
	})

	if errDocker != nil {
		t.Skipf("no Docker on this host: %v", errDocker)
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
