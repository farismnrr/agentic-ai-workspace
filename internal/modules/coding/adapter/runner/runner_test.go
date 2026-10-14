package runner_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func run(t *testing.T, seconds int, command ...string) application.RunResult {
	t.Helper()

	result, err := runner.New(t.TempDir()).Run(context.Background(),
		domain.Task{Name: "t", Command: command, TimeoutSeconds: seconds})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestReportsExitCodeAndOutput(t *testing.T) {
	t.Parallel()

	result := run(t, 10, "sh", "-c", "echo hello; echo oops >&2; exit 3")
	if result.ExitCode != 3 || !strings.Contains(result.Output, "hello") || !strings.Contains(result.Output, "oops") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTimeoutKillsTheWholeProcessGroup(t *testing.T) {
	t.Parallel()

	started := time.Now()
	result := run(t, 1, "sh", "-c", "sleep 30 & sleep 30")

	if !result.TimedOut || time.Since(started) > 10*time.Second {
		t.Fatalf("task should be killed near its timeout: %+v after %v", result, time.Since(started))
	}
}

func TestOutputKeepsOnlyTheTail(t *testing.T) {
	t.Parallel()

	result := run(t, 10, "sh", "-c", "head -c 200000 /dev/zero | tr '\\0' x; echo END")
	if !result.Truncated || len(result.Output) > domain.MaxOutputBytes || !strings.HasSuffix(result.Output, "END\n") {
		t.Fatalf("expected a capped tail, got %d bytes truncated=%v", len(result.Output), result.Truncated)
	}
}

func TestEnvironmentIsScrubbed(t *testing.T) {
	t.Setenv("OAUTH_SECRET_FOR_TEST", "leak")

	result := run(t, 10, "env")
	if strings.Contains(result.Output, "OAUTH_SECRET_FOR_TEST") || !strings.Contains(result.Output, "PATH=") {
		t.Fatalf("only build-tool variables may reach a task:\n%s", result.Output)
	}
}

func TestMissingCommandIsReportedNotFatal(t *testing.T) {
	t.Parallel()

	result := run(t, 5, "definitely-not-a-command-xyz")
	if result.ExitCode != -1 || !strings.Contains(result.Output, "could not start") {
		t.Fatalf("got %+v", result)
	}
}

func TestOnlyOneTaskRunsAtATime(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := runner.New(dir)
	done := make(chan struct{})

	go func() {
		_, _ = r.Run(context.Background(), domain.Task{
			Command: []string{"sh", "-c", "touch started; sleep 3"}, TimeoutSeconds: 10,
		})
		close(done)
	}()

	// Wait until the first task is really running, so the check does not depend on timing.
	for range 100 {
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
