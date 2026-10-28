package runner_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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

var errSnapshotTestNoDocker = errors.New("snapshot regression does not require Docker")

type snapshotRun struct {
	err   error
	value application.RunResult
}

func TestRunnerSnapshotSymlinkToFIFOIsBoundedWithoutDocker(t *testing.T) {
	workspace := t.TempDir()
	pipe := filepath.Join(workspace, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("pipe", filepath.Join(workspace, "link")); err != nil {
		t.Fatal(err)
	}
	engine := runner.NewEngine(runner.EngineConfig{
		Size: 1,
		Open: func(context.Context) (*sandbox.Docker, error) {
			return nil, errSnapshotTestNoDocker
		},
	})
	started := time.Now()
	result := runSnapshot(workspace, engine)
	awaitBoundedSnapshot(t, result, pipe, started)
}

func runSnapshot(workspace string, engine *runner.Engine) <-chan snapshotRun {
	result := make(chan snapshotRun, 1)
	go func() {
		value, err := runner.New(workspace, engine).Run(context.Background(),
			domain.Task{Name: "snapshot", Command: []string{"true"}, TimeoutSeconds: 1})
		result <- snapshotRun{err: err, value: value}
	}()
	return result
}

func awaitBoundedSnapshot(t *testing.T, result <-chan snapshotRun, pipe string, started time.Time) {
	t.Helper()
	select {
	case outcome := <-result:
		if outcome.err != nil || outcome.value.EffectsKnown || time.Since(started) >= time.Second {
			t.Fatalf("snapshot must return promptly and conservatively before Docker: %+v, %v", outcome.value, outcome.err)
		}
	case <-time.After(time.Second):
		fd, err := syscall.Open(pipe, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatal(err)
		}
		_ = syscall.Close(fd)
		select {
		case <-result:
			t.Fatal("Runner.Run blocked opening a symlink to a FIFO")
		case <-time.After(time.Second):
			t.Fatal("Runner.Run remained blocked after releasing the FIFO")
		}
	}
}
