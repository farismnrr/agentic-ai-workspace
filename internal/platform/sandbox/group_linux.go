package sandbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Group is one task's cgroup. The task is started inside it, so it can never run
// outside its limits, and Close kills whatever is left.
type Group struct {
	path string
	fd   int
}

// New creates a cgroup holding the limits.
func (c *Cgroups) New(limits Limits) (*Group, error) {
	suffix := make([]byte, suffixBytes)
	if _, err := rand.Read(suffix); err != nil {
		return nil, fmt.Errorf("name cgroup: %w", err)
	}

	path := c.parent + "/task-" + hex.EncodeToString(suffix)
	if err := os.Mkdir(path, dirModeCgroup); err != nil {
		return nil, fmt.Errorf("create cgroup: %w", err)
	}

	settings := []struct{ file, value string }{
		{"memory.max", strconv.FormatUint(limits.MemoryBytes, base)},
		{"memory.swap.max", "0"},
		{"memory.oom.group", "1"},
		{"pids.max", strconv.FormatUint(limits.Processes, base)},
		{"cpu.max", strconv.FormatUint(limits.CPUMillis*cpuPeriod/millisPerCore, base) + " " + strconv.Itoa(cpuPeriod)},
	}

	for _, setting := range settings {
		if err := write(path+"/"+setting.file, setting.value); err != nil {
			_ = os.Remove(path)

			return nil, err
		}
	}

	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = os.Remove(path)

		return nil, fmt.Errorf("open cgroup: %w", err)
	}

	return &Group{path: path, fd: fd}, nil
}

// Path is the cgroup's folder, for inspection.
func (g *Group) Path() string { return g.path }

// FD is the descriptor the kernel needs to start a process inside the cgroup.
func (g *Group) FD() int { return g.fd }

// OOMKilled reports whether the kernel killed the task for exceeding its memory limit.
func (g *Group) OOMKilled() bool {
	data, err := os.ReadFile(g.path + "/memory.events")
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(data), "\n") {
		if name, count, ok := strings.Cut(line, " "); ok && name == "oom_kill" {
			return strings.TrimSpace(count) != "0"
		}
	}

	return false
}

// Close kills anything still running in the cgroup and removes it.
func (g *Group) Close() {
	_ = write(g.path+"/cgroup.kill", "1")
	_ = unix.Close(g.fd)

	deadline := time.Now().Add(killWait)
	for time.Now().Before(deadline) {
		if os.Remove(g.path) == nil {
			return
		}

		time.Sleep(killPoll)
	}
}

func write(path, value string) error {
	//nolint:gosec // paths are built from the server's own delegated cgroup
	if err := os.WriteFile(path, []byte(value), fileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
