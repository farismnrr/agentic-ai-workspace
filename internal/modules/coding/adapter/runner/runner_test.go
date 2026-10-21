package runner_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestReportsExitCodeAndOutput(t *testing.T) {
	t.Parallel()

	result := run(t, 30, "sh", "-c", "echo hello; echo oops >&2; exit 3")
	if result.ExitCode != 3 || !strings.Contains(result.Output, "hello") || !strings.Contains(result.Output, "oops") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTimeoutStopsTheWholeContainer(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	started := time.Now()
	result := run(t, 1, "sh", "-c", "sleep 30 & sleep 30")

	if !result.TimedOut || time.Since(started) > 15*time.Second {
		t.Fatalf("task should be stopped near its timeout: %+v after %v", result, time.Since(started))
	}
}

func TestOutputKeepsOnlyTheTail(t *testing.T) {
	t.Parallel()

	result := run(t, 30, "sh", "-c", "head -c 200000 /dev/zero | tr '\\0' x; echo END")
	if !result.Truncated || len(result.Output) > domain.MaxOutputBytes || !strings.HasSuffix(result.Output, "END\n") {
		t.Fatalf("expected a capped tail, got %d bytes truncated=%v", len(result.Output), result.Truncated)
	}
}

func TestEnvironmentIsScrubbed(t *testing.T) {
	t.Setenv("OAUTH_SECRET_FOR_TEST", "leak")

	result := run(t, 30, "env")
	if strings.Contains(result.Output, "OAUTH_SECRET_FOR_TEST") || !strings.Contains(result.Output, "PATH=") {
		t.Fatalf("only build-tool variables may reach a task:\n%s", result.Output)
	}
}

type projectEnvValues map[string]map[string]string

func (v projectEnvValues) Values(_ context.Context, projectID string) (map[string]string, error) {
	return v[projectID], nil
}

func TestProjectCommandReceivesOnlyItsProjectValues(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	values := projectEnvValues{
		"project-a": {"PROJECT_TOKEN": "a-only"},
		"project-b": {"PROJECT_TOKEN": "b-only"},
	}
	for projectID, expected := range map[string]string{"project-a": "a-only", "project-b": "b-only"} {
		result, err := runner.New(t.TempDir(), engine).WithProjectEnv(projectID, values).Run(context.Background(),
			domain.Task{Command: []string{"sh", "-c", "printf '%s' \"$PROJECT_TOKEN\""}, TimeoutSeconds: 30})
		if err != nil || result.ExitCode != 0 || result.Output != expected {
			t.Fatalf("project %s command output = %q, %v (exit %d)", projectID, result.Output, err, result.ExitCode)
		}
	}
}

func TestMissingCommandIsReportedNotFatal(t *testing.T) {
	t.Parallel()

	result := run(t, 30, "definitely-not-a-command-xyz")
	if result.ExitCode == 0 || !strings.Contains(result.Output, "No such file") {
		t.Fatalf("got %+v", result)
	}
}

func TestOnlyOneTaskRunsAtATime(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	dir := t.TempDir()
	r := runner.New(dir, engine)
	done := make(chan struct{})

	go func() {
		_, _ = r.Run(context.Background(), domain.Task{
			Command: []string{"sh", "-c", "touch /workspace/started; sleep 3"}, TimeoutSeconds: 30,
		})
		close(done)
	}()

	// Wait until the first task is really running, so the check does not depend on timing.
	for range 200 {
		if _, err := os.Stat(filepath.Join(dir, "started")); err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	_, err := r.Run(context.Background(), domain.Task{Command: []string{"true"}})
	if !errors.Is(err, domain.ErrTaskBusy) {
		t.Fatalf("a second task must be refused while one is running, got %v", err)
	}

	<-done
}
