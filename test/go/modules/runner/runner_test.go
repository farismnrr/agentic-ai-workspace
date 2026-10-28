package runner_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// testImage is small and has the shell tools these tests need.
const testImage = "busybox:1.37"

// engine is shared by the tests like the server shares one between all workspaces.
var (
	engine      = newEngine(8, sandbox.DefaultLimits())
	dockerOnce  sync.Once
	errDocker   error
	probeReason = "no Docker daemon on this host"
)

// newEngine builds an engine on the test image with room for size containers.
func newEngine(size int, limits sandbox.Limits) *runner.Engine {
	return runner.NewEngine(runner.EngineConfig{
		Size: size,
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			return sandbox.Open(ctx, sandbox.Config{
				Image: testImage, Limits: limits, UID: os.Getuid(), GID: os.Getgid(),
			})
		},
	})
}

// requireDocker skips a test that runs a container when the host has no usable Docker.
func requireDocker(t *testing.T) {
	t.Helper()

	dockerOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		_, errDocker = engine.Docker(ctx)
	})

	if errDocker != nil {
		t.Skipf("%s: %v", probeReason, errDocker)
	}
}

func runWith(t *testing.T, eng *runner.Engine, seconds int, command ...string) application.RunResult {
	t.Helper()
	requireDocker(t)

	result, err := runner.New(t.TempDir(), eng).Run(context.Background(),
		domain.Task{Name: "t", Command: command, TimeoutSeconds: seconds})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func run(t *testing.T, seconds int, command ...string) application.RunResult {
	t.Helper()

	return runWith(t, engine, seconds, command...)
}
