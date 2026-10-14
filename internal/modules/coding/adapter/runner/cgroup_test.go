package runner_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// cgroupEngine needs cgroups delegated to the test process (run it under
// `systemd-run --user --scope -p Delegate=yes`); elsewhere these tests skip.
func cgroupEngine(t *testing.T, limits sandbox.Limits) *runner.Engine {
	t.Helper()

	if errToolbox != nil {
		t.Skipf("no sandbox toolbox on this host: %v", errToolbox)
	}

	cgroups, err := sandbox.OpenCgroups()
	if err != nil {
		t.Skipf("cgroups are not delegated to this process: %v", err)
	}

	return runner.NewEngine(runner.EngineConfig{
		Cgroups: cgroups, Toolbox: filepath.Join(sandboxDir, "general"), Limits: limits, Size: 4,
	})
}

func runIn(t *testing.T, e *runner.Engine, seconds int, command ...string) (string, int, bool) {
	t.Helper()

	result, err := runner.New(t.TempDir(), e).Run(context.Background(),
		domain.Task{Name: "t", Command: command, TimeoutSeconds: seconds})
	if err != nil {
		t.Fatal(err)
	}

	return result.Output, result.ExitCode, result.TimedOut
}

func TestACgroupHoldsTheTaskToItsLimits(t *testing.T) {
	limits := sandbox.DefaultLimits()
	limits.MemoryBytes, limits.Processes, limits.CPUMillis = 256<<20, 64, 250
	group, err := openGroup(t, limits)
	if err != nil {
		t.Skip(err)
	}

	want := map[string]string{
		"memory.max": "268435456", "memory.swap.max": "0", "pids.max": "64", "cpu.max": "25000 100000",
		"memory.oom.group": "1",
	}

	for file, value := range want {
		data, readErr := os.ReadFile(filepath.Join(group.Path(), file)) //nolint:gosec // a cgroup this test created
		if readErr != nil || strings.TrimSpace(string(data)) != value {
			t.Errorf("%s = %q (%v), want %q", file, data, readErr, value)
		}
	}

	path := group.Path()
	group.Close()

	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("the cgroup must be removed when the task is over")
	}
}

func openGroup(t *testing.T, limits sandbox.Limits) (*sandbox.Group, error) {
	t.Helper()

	cgroups, err := sandbox.OpenCgroups()
	if err != nil {
		return nil, err //nolint:wrapcheck // only used to decide to skip
	}

	group, err := cgroups.New(limits)
	if err != nil {
		return nil, fmt.Errorf("create cgroup: %w", err)
	}

	return group, nil
}

func TestMemoryIsEnforcedByTheKernelAndReported(t *testing.T) {
	limits := sandbox.DefaultLimits()
	limits.MemoryBytes = 128 << 20
	limits.Processes = 64
	e := cgroupEngine(t, limits)

	// Touching 400 MB of fresh memory is far over the 128 MB limit.
	script := `BEGIN { s = sprintf("%1000000s", ""); for (i = 0; i < 400; i++) a[i] = s; print "allocated" }`
	output, code, _ := runIn(t, e, 60, "awk", script)

	if strings.Contains(output, "allocated") || code == 0 || !strings.Contains(output, "more memory than allowed") {
		t.Fatalf("the kernel must stop it and the person must be told why (exit %d):\n%s", code, output)
	}
}

func TestProcessesAreCappedSoAForkBombCannotSpread(t *testing.T) {
	limits := sandbox.DefaultLimits()
	limits.Processes = 24
	e := cgroupEngine(t, limits)

	output, _, timedOut := runIn(t, e, 20, "sh", "-c", "for i in $(seq 1 200); do sleep 3 & done; wait")

	if timedOut || !strings.Contains(strings.ToLower(output), "can't fork") {
		t.Fatalf("a task must hit its process cap instead of spreading, got:\n%s", output)
	}
}
