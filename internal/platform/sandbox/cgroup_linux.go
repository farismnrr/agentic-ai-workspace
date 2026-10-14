package sandbox

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// ErrNoCgroups means the server cannot create cgroups, so it cannot enforce real
// CPU, memory and process limits. Run it as a systemd service with Delegate=yes.
var ErrNoCgroups = errors.New("cgroup v2 limits are not available")

var errControllerMissing = errors.New("a needed controller is not delegated to the server")

const (
	cgroupRoot     = "/sys/fs/cgroup"
	supervisorName = "supervisor"
	cpuPeriod      = 100_000
	millisPerCore  = 1000
	killWait       = 3 * time.Second
	killPoll       = 20 * time.Millisecond
	fileMode       = 0o644
	dirModeCgroup  = 0o755
	suffixBytes    = 6
)

var needed = []string{"cpu", "memory", "pids"}

// Cgroups hands out one cgroup per sandboxed task under the cgroup the server was
// delegated. It is the same mechanism rootless container runtimes use.
type Cgroups struct {
	parent string
}

// The server's own cgroup is process-wide, so it is prepared once and the outcome is reused.
var (
	openOnce   sync.Once
	openResult *Cgroups
	errOpen    error
)

// OpenCgroups prepares the delegated cgroup (once per process): the server moves into a child of its own
// cgroup (v2 forbids a cgroup from holding processes and delegating controllers at
// once) and enables the cpu, memory and pids controllers for the tasks it creates.
func OpenCgroups() (*Cgroups, error) {
	openOnce.Do(func() { openResult, errOpen = prepareCgroups() })

	return openResult, errOpen
}

func prepareCgroups() (*Cgroups, error) {
	own, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoCgroups, err)
	}

	_, rel, found := strings.Cut(strings.TrimSpace(string(own)), "::")
	if !found {
		return nil, fmt.Errorf("%w: this host does not use cgroup v2", ErrNoCgroups)
	}

	parent := cgroupRoot + rel
	if err := unix.Access(parent, unix.W_OK); err != nil {
		return nil, fmt.Errorf("%w: %s is not writable by the server (run it as a systemd service with Delegate=yes)",
			ErrNoCgroups, parent)
	}

	if err := delegate(parent); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoCgroups, err)
	}

	return &Cgroups{parent: parent}, nil
}

func delegate(parent string) error {
	//nolint:gosec // parent is the server's own delegated cgroup
	available, err := os.ReadFile(parent + "/cgroup.controllers")
	if err != nil {
		return fmt.Errorf("read controllers: %w", err)
	}

	for _, name := range needed {
		if !strings.Contains(" "+strings.TrimSpace(string(available))+" ", " "+name+" ") {
			return fmt.Errorf("%w: %s", errControllerMissing, name)
		}
	}

	supervisor := parent + "/" + supervisorName
	//nolint:gosec // supervisor is under the server's own delegated cgroup
	if err = os.MkdirAll(supervisor, dirModeCgroup); err != nil {
		return fmt.Errorf("create %s: %w", supervisor, err)
	}

	if err = write(supervisor+"/cgroup.procs", strconv.Itoa(os.Getpid())); err != nil {
		return err
	}

	if err = write(parent+"/cgroup.subtree_control", "+cpu +memory +pids"); err != nil {
		return fmt.Errorf("enable controllers (another process shares the server's cgroup?): %w", err)
	}

	return nil
}
