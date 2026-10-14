package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const base = 10

// processBackstop caps forks when no cgroup does. The kernel counts every process
// of the host user, so it only stops runaway fork bombs; it is skipped under a cgroup.
const processBackstop = 4096

// Command prepares spec.Argv to run inside a fresh sandbox. The returned cleanup
// must be called after the command ends.
func Command(ctx context.Context, spec Spec) (*exec.Cmd, func(), error) {
	if err := ToolboxReady(spec.Toolbox); err != nil {
		return nil, nil, err
	}

	newRoot, err := os.MkdirTemp("", "sandbox-")
	if err != nil {
		return nil, nil, fmt.Errorf("prepare sandbox: %w", err)
	}

	// Under a cgroup the kernel counts memory and processes exactly; the coarser rlimits
	// that stand in for them without one (value 0 skips them) would only muddy the reports.
	memory, processes := spec.Limits.MemoryBytes, uint64(processBackstop)
	if spec.Group != nil {
		memory, processes = 0, 0
	}

	numbers := []string{
		strconv.FormatUint(memory, base), strconv.FormatUint(spec.Limits.FileBytes, base),
		strconv.FormatUint(processes, base),
	}
	args := []string{initFlag, newRoot, spec.Toolbox, spec.Workspace}
	args = append(args, numbers...)
	args = append(args, "--")
	args = append(args, spec.Argv...)

	//nolint:gosec // re-executing this binary as the sandbox init is the design
	cmd := exec.CommandContext(ctx, "/proc/self/exe", args...)
	cmd.Env = spec.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
	}

	if spec.Group != nil {
		cmd.SysProcAttr.UseCgroupFD, cmd.SysProcAttr.CgroupFD = true, spec.Group.FD()
	}

	// The init is PID 1 of its namespace: killing it takes every process with it.
	cmd.Cancel = func() error { return cmd.Process.Kill() }

	return cmd, func() { _ = os.Remove(newRoot) }, nil
}
