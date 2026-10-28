package runner_test

import (
	"strings"
	"testing"
)

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
