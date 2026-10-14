package sandbox

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var errBadArguments = errors.New("sandbox init was started with bad arguments")

const (
	dirMode    = 0o755
	stubMode   = 0o400
	inheriting = unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC | unix.MS_NOATIME |
		unix.MS_NODIRATIME | unix.MS_RELATIME
)

// deviceFiles are the only host files a sandbox sees: the harmless character devices
// every program expects. They are bound in because an unprivileged server cannot create devices.
var deviceFiles = []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom"}

// buildRoot assembles the private root and makes it the root: the toolbox, read-only,
// with a private /tmp, a minimal /dev and /proc, and the workspace on top.
func buildRoot(newRoot, toolbox, workspace string) error {
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}

	// The toolbox is the root. Its mount points (/proc, /dev, /tmp, /workspace) were made
	// when it was installed, because nothing can be created on a read-only root.
	if err := bind(toolbox, newRoot, true); err != nil {
		return err
	}

	if err := bind(workspace, newRoot+Workspace, false); err != nil {
		return err
	}

	if err := mountScratch(newRoot); err != nil {
		return err
	}

	return pivot(newRoot)
}

func mountScratch(newRoot string) error {
	tmp, dev := newRoot+"/tmp", newRoot+"/dev"
	flags := uintptr(unix.MS_NOSUID | unix.MS_NODEV)

	if err := unix.Mount("tmpfs", tmp, "tmpfs", flags, "mode=1777,size=512m"); err != nil {
		return fmt.Errorf("mount /tmp: %w", err)
	}

	if err := unix.Mount("tmpfs", dev, "tmpfs", flags, "mode=755,size=1m"); err != nil {
		return fmt.Errorf("mount /dev: %w", err)
	}

	for _, node := range deviceFiles {
		if err := bind(node, newRoot+node, false); err != nil {
			return err
		}
	}

	proc := uintptr(unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC)
	if err := unix.Mount("proc", newRoot+"/proc", "proc", proc, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}

	return nil
}

// pivot makes newRoot the root and enters the workspace.
func pivot(newRoot string) error {
	if err := swapRoot(newRoot); err != nil {
		return err
	}

	if err := unix.Chdir(Workspace); err != nil {
		return fmt.Errorf("enter workspace: %w", err)
	}

	return nil
}

// swapRoot swaps the root for newRoot and drops every trace of the host's tree.
func swapRoot(newRoot string) error {
	if err := unix.Chdir(newRoot); err != nil {
		return fmt.Errorf("enter new root: %w", err)
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return fmt.Errorf("detach old root: %w", err)
	}

	return nil
}

// mkdir creates dir (and parents) under a root this process just made.
func mkdir(dir string) error {
	//nolint:gosec // dir is under a private root this process just created
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	return nil
}
