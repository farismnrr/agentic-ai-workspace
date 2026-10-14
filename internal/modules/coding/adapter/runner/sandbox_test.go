package runner_test

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// The sandbox needs unprivileged user namespaces; hosts that forbid them (some
// CI images) skip these checks instead of failing.
func requireSandbox(t *testing.T) {
	t.Helper()

	if result := run(t, 10, "true"); result.ExitCode != 0 {
		t.Skipf("sandbox unavailable on this host: %s", result.Output)
	}
}

func TestTaskSeesItsOwnToolboxNotTheHost(t *testing.T) {
	requireSandbox(t)

	secret := filepath.Join(t.TempDir(), "host-secret")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if result := run(t, 10, "cat", secret); result.ExitCode == 0 {
		t.Errorf("a host file must not be readable, output: %s", result.Output)
	}

	// The toolbox has its own accounts; the host's user does not exist in it, and no
	// host folder shows up under the usual places.
	me, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	result := run(t, 10, "sh", "-c", "cat /etc/passwd; ls /home /root /mnt /media")
	if strings.Contains(result.Output, me.Username) || strings.Contains(result.Output, "agentic") {
		t.Errorf("the host's accounts and folders must not be visible:\n%s", result.Output)
	}
}

func TestOnlyTheWorkspaceIsWritable(t *testing.T) {
	requireSandbox(t)

	result := run(t, 10, "sh", "-c", "touch /usr/x 2>&1; touch /etc/x 2>&1; touch /workspace/ok && echo wrote")
	if !strings.Contains(result.Output, "wrote") || strings.Count(result.Output, "Read-only") != 2 {
		t.Fatalf("expected /usr and /etc read-only and /workspace writable:\n%s", result.Output)
	}
}

func TestTaskHasNoNetworkNoOtherProcessesNoPrivileges(t *testing.T) {
	requireSandbox(t)

	script := "tail -n +3 /proc/net/dev | cut -d: -f1 | tr -d ' '; ls /proc | grep -c '^[0-9]'; " +
		"grep -E '^(CapEff|NoNewPrivs)' /proc/self/status"

	result := run(t, 10, "sh", "-c", script)
	for _, want := range []string{"lo\n", "CapEff:\t0000000000000000", "NoNewPrivs:\t1"} {
		if !strings.Contains(result.Output, want) {
			t.Errorf("missing %q in:\n%s", want, result.Output)
		}
	}
}

func TestDangerousSystemCallsAreRefused(t *testing.T) {
	requireSandbox(t)

	// unshare asks the kernel for a new namespace, which the filter forbids even though
	// user namespaces are what the sandbox itself is built from.
	result := run(t, 10, "sh", "-c", "unshare -U true 2>&1; unshare -m true 2>&1; echo still-works")
	if strings.Count(result.Output, "not permitted") != 2 || !strings.Contains(result.Output, "still-works") {
		t.Fatalf("namespace creation must fail while ordinary commands keep working:\n%s", result.Output)
	}
}

func TestMemoryHogsAreStopped(t *testing.T) {
	requireSandbox(t)

	// 4 GB of strings is over the 2 GiB limit, so the program must fail instead of finishing.
	script := `BEGIN { s = sprintf("%1000000s", ""); for (i = 0; i < 4000; i++) a[i] = s; print "allocated" }`
	result := run(t, 60, "awk", script)

	if strings.Contains(result.Output, "allocated") || result.ExitCode == 0 {
		t.Fatalf("a program that grabs 4 GB must be stopped:\n%s", result.Output)
	}
}

func TestOversizedFilesAreRefused(t *testing.T) {
	requireSandbox(t)

	script := "head -c 700000000 /dev/zero > /workspace/big 2>&1; ls -l /workspace/big | tr -s ' ' | cut -d' ' -f5"
	result := run(t, 30, "sh", "-c", script)

	if size := strings.TrimSpace(result.Output); size == "700000000" {
		t.Fatalf("one file may not exceed the file size limit, got %s bytes", size)
	}
}

func TestAWorkspaceThatGrowsTooLargeStopsTheTask(t *testing.T) {
	requireSandbox(t)

	r := runner.New(t.TempDir(), engine).WithWorkspaceLimit(1 << 20)
	result, err := r.Run(context.Background(), domain.Task{
		Name: "fill", TimeoutSeconds: 30,
		Command: []string{"sh", "-c", "while true; do head -c 400000 /dev/zero >> /workspace/f$RANDOM; sleep 1; done"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.TimedOut || result.ExitCode != -1 || !strings.Contains(result.Output, "size limit") {
		t.Fatalf("the task must be stopped for filling the workspace: %+v", result)
	}
}

func TestAnAlreadyFullWorkspaceDoesNotStartATask(t *testing.T) {
	requireSandbox(t)

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

func TestOnlyAFewSandboxesRunAtOnce(t *testing.T) {
	requireSandbox(t)

	small := newEngine(1)
	first, second := runner.New(t.TempDir(), small), runner.New(t.TempDir(), small)
	finished := make(chan struct{})

	go func() {
		_, _ = first.Run(context.Background(), domain.Task{
			Command: []string{"sh", "-c", "sleep 2"}, TimeoutSeconds: 10,
		})
		close(finished)
	}()

	time.Sleep(500 * time.Millisecond)

	begin := time.Now()
	if _, err := second.Run(context.Background(), domain.Task{Command: []string{"true"}, TimeoutSeconds: 10}); err != nil {
		t.Fatal(err)
	}

	if time.Since(begin) < time.Second {
		t.Fatal("the second sandbox must wait for the pool, not run alongside the first")
	}

	<-finished
}

func TestWithoutAToolboxNothingRunsAndTheOwnerIsToldWhatToDo(t *testing.T) {
	t.Parallel()

	bare := runner.NewEngine(runner.EngineConfig{Toolbox: t.TempDir(), Size: 1})
	result, err := runner.New(t.TempDir(), bare).Run(context.Background(),
		domain.Task{Name: "t", TimeoutSeconds: 10, Command: []string{"true"}})

	if err != nil || result.ExitCode != -1 || !strings.Contains(result.Output, "sandbox install") {
		t.Fatalf("a missing toolbox must say how to install it: %+v %v", result, err)
	}
}
