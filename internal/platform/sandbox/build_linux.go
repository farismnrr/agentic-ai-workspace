package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// buildFlag marks a re-executed copy of this binary as the toolbox installer's init.
const buildFlag = "--sandbox-build"

// buildCommand prepares argv to run as root inside the toolbox being built. Unlike a task
// sandbox it keeps the host network (installing packages needs it) and its capabilities
// (package files are owned by "root"), and the toolbox is writable. It is only ever run
// by the server's owner, from the install command, never for a person's code.
func buildCommand(ctx context.Context, stage string, argv []string, out io.Writer) *exec.Cmd {
	args := append([]string{buildFlag, stage, "--"}, argv...)

	//nolint:gosec // re-executing this binary as the installer's init is the design
	cmd := exec.CommandContext(ctx, "/proc/self/exe", args...)
	cmd.Stdout, cmd.Stderr = out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
	}

	return cmd
}

// runBuild is the installer's init: it enters the toolbox and runs the command in it.
func runBuild(args []string) {
	runtime.LockOSThread()

	if err := enterBuild(args); err != nil {
		fmt.Fprintln(os.Stderr, "toolbox build:", err)
		os.Exit(failureStatus)
	}
}

func enterBuild(args []string) error {
	if len(args) < 3 || args[1] != "--" {
		return errBadArguments
	}

	stage, command := args[0], args[2:]

	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}

	if err := unix.Mount(stage, stage, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("bind toolbox: %w", err)
	}

	proc := uintptr(unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC)
	if err := unix.Mount("proc", stage+"/proc", "proc", proc, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}

	for _, node := range deviceFiles {
		if err := bind(node, stage+node, false); err != nil {
			return err
		}
	}

	if err := swapRoot(stage); err != nil {
		return err
	}

	env := []string{"PATH=" + Path, "HOME=/root", "TERM=dumb", "LANG=C.UTF-8"}

	//nolint:gosec // the command is the installer's fixed package-manager call
	if err := syscall.Exec(command[0], command, env); err != nil {
		return fmt.Errorf("start %s: %w", command[0], err)
	}

	return nil
}
