package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func TestTaskDoesNotSeeTheHost(t *testing.T) {
	t.Parallel()

	secret := filepath.Join(t.TempDir(), "host-secret")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if result := run(t, 30, "cat", secret); result.ExitCode == 0 {
		t.Errorf("a host file must not be readable, output: %s", result.Output)
	}
}

func TestOnlyTheWorkspaceIsWritable(t *testing.T) {
	t.Parallel()

	result := run(t, 30, "sh", "-c", "touch /usr/x 2>&1; touch /etc/x 2>&1; touch /workspace/ok && echo wrote")
	if !strings.Contains(result.Output, "wrote") || strings.Count(result.Output, "Read-only") != 2 {
		t.Fatalf("expected /usr and /etc read-only and /workspace writable:\n%s", result.Output)
	}
}

func TestTaskHasNoNetworkAndNoPrivileges(t *testing.T) {
	t.Parallel()

	script := "ls /sys/class/net; grep -E '^(CapEff|NoNewPrivs)' /proc/self/status"

	result := run(t, 30, "sh", "-c", script)
	for _, want := range []string{"lo\n", "CapEff:\t0000000000000000", "NoNewPrivs:\t1"} {
		if !strings.Contains(result.Output, want) {
			t.Errorf("missing %q in:\n%s", want, result.Output)
		}
	}

	if strings.Contains(result.Output, "eth0") {
		t.Errorf("a task must have no network interface:\n%s", result.Output)
	}
}

func TestMemoryHogsAreStopped(t *testing.T) {
	t.Parallel()

	limits := sandbox.DefaultLimits()
	limits.MemoryBytes = 128 << 20

	// 600 MB of strings is far over the 128 MiB limit, so the program must not finish.
	script := `BEGIN { s = sprintf("%1000000s", ""); for (i = 0; i < 600; i++) a[i] = s; print "allocated" }`
	result := runWith(t, newEngine(1, limits), 60, "awk", script)

	if strings.Contains(result.Output, "allocated") || result.ExitCode == 0 {
		t.Fatalf("a program that grabs 600 MB must be stopped:\n%s", result.Output)
	}
}

func TestOversizedFilesAreRefused(t *testing.T) {
	t.Parallel()

	limits := sandbox.DefaultLimits()
	limits.FileBytes = 4 << 20

	script := "head -c 20000000 /dev/zero > /workspace/big 2>&1; ls -l /workspace/big | tr -s ' ' | cut -d' ' -f5"
	result := runWith(t, newEngine(1, limits), 30, "sh", "-c", script)

	if size := strings.TrimSpace(result.Output); size == "20000000" {
		t.Fatalf("one file may not exceed the file size limit, got %s bytes", size)
	}
}

func TestAWorkspaceThatGrowsTooLargeStopsTheTask(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	r := runner.New(t.TempDir(), engine).WithWorkspaceLimit(1 << 20)
	grow := "while true; do head -c 400000 /dev/zero >> /workspace/f; sleep 1; done"
	result, err := r.Run(context.Background(), domain.Task{
		Name: "fill", TimeoutSeconds: 60,
		Command: []string{"sh", "-c", grow},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.TimedOut || result.ExitCode != -1 || !strings.Contains(result.Output, "size limit") {
		t.Fatalf("the task must be stopped for filling the workspace: %+v", result)
	}
}

func TestAnAlreadyFullWorkspaceDoesNotStartATask(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "big"), make([]byte, 2<<20), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := runner.New(dir, engine).WithWorkspaceLimit(1<<20).Run(context.Background(),
		domain.Task{Name: "t", TimeoutSeconds: 10, Command: []string{"true"}})
	if err != nil || result.ExitCode != -1 || !strings.Contains(result.Output, "size limit") {
		t.Fatalf("a full workspace must be refused up front: %+v %v", result, err)
	}
}

func TestOnlyAFewContainersRunAtOnce(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	small := newEngine(1, sandbox.DefaultLimits())
	first, second := runner.New(t.TempDir(), small), runner.New(t.TempDir(), small)
	finished := make(chan struct{})

	go func() {
		_, _ = first.Run(context.Background(), domain.Task{
			Command: []string{"sh", "-c", "sleep 3"}, TimeoutSeconds: 30,
		})
		close(finished)
	}()

	time.Sleep(time.Second)

	begin := time.Now()
	if _, err := second.Run(context.Background(), domain.Task{Command: []string{"true"}, TimeoutSeconds: 30}); err != nil {
		t.Fatal(err)
	}

	if time.Since(begin) < time.Second {
		t.Fatal("the second container must wait for the pool, not run alongside the first")
	}

	<-finished
}

func TestWithoutDockerNothingRunsAndTheReasonIsPlain(t *testing.T) {
	t.Parallel()

	bare := runner.NewEngine(runner.EngineConfig{
		Size: 1,
		Open: func(context.Context) (*sandbox.Docker, error) { return nil, sandbox.ErrUnavailable },
	})
	result, err := runner.New(t.TempDir(), bare).Run(context.Background(),
		domain.Task{Name: "t", TimeoutSeconds: 10, Command: []string{"true"}})

	if err != nil || result.ExitCode != -1 || !strings.Contains(result.Output, "docker is not available") {
		t.Fatalf("a missing Docker must be said plainly: %+v %v", result, err)
	}
}
