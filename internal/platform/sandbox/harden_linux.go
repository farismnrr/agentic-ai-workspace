package sandbox

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// harden removes what the command must never have and applies the limits. It runs
// after the root is in place and just before exec, in this order: limits, a loopback
// network, no new privileges, no capabilities, and last the system call filter.
func harden(limits Limits) error {
	if err := unix.Sethostname([]byte("sandbox")); err != nil {
		return fmt.Errorf("set hostname: %w", err)
	}

	if err := loopbackUp(); err != nil {
		return err
	}

	if err := applyLimits(limits); err != nil {
		return err
	}

	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("forbid new privileges: %w", err)
	}

	if err := dropCapabilities(); err != nil {
		return err
	}

	return installFilter()
}

func applyLimits(limits Limits) error {
	// A zero memory or process limit means "a cgroup does this": the rlimit is skipped.
	wanted := []rule{{"file size", unix.RLIMIT_FSIZE, limits.FileBytes}, {"core dumps", unix.RLIMIT_CORE, 0}}
	if limits.MemoryBytes != 0 {
		wanted = append(wanted, rule{"memory", unix.RLIMIT_DATA, limits.MemoryBytes})
	}

	if limits.Processes != 0 {
		wanted = append(wanted, rule{"processes", unix.RLIMIT_NPROC, limits.Processes})
	}

	for _, item := range wanted {
		if err := unix.Setrlimit(item.resource, &unix.Rlimit{Cur: item.value, Max: item.value}); err != nil {
			return fmt.Errorf("limit %s: %w", item.name, err)
		}
	}

	return nil
}

type rule struct {
	name     string
	resource int
	value    uint64
}

// dropCapabilities empties the bounding set, so no exec can regain a capability.
func dropCapabilities() error {
	for capability := 0; ; capability++ {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(capability), 0, 0, 0)
		if errors.Is(err, unix.EINVAL) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("drop capability %d: %w", capability, err)
		}
	}
}

// loopbackUp lets a command talk to itself (test servers) without any real network.
func loopbackUp() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open socket: %w", err)
	}

	defer func() { _ = unix.Close(fd) }()

	request, err := unix.NewIfreq("lo")
	if err != nil {
		return fmt.Errorf("name interface: %w", err)
	}

	request.SetUint16(unix.IFF_UP)

	if err := unix.IoctlIfreq(fd, unix.SIOCSIFFLAGS, request); err != nil {
		return fmt.Errorf("bring up loopback: %w", err)
	}

	return nil
}
