package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// bind exposes src at dst, optionally read-only. The mount point is created when
// the filesystem under it allows it; on a read-only toolbox it must already exist.
func bind(src, dst string, readOnly bool) error {
	//nolint:gosec // src is the toolbox, the workspace or a fixed device path
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", src, err)
	}

	if err = makeTarget(dst, info.IsDir()); err != nil {
		return err
	}

	if err = unix.Mount(src, dst, "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind %s: %w", src, err)
	}

	if readOnly {
		return remountReadOnly(dst)
	}

	return nil
}

// remountReadOnly seals a bind mount, keeping the flags its source already carries
// (the kernel rejects a remount that would loosen them inside a user namespace).
func remountReadOnly(dst string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(dst, &stat); err != nil {
		return fmt.Errorf("inspect mount %s: %w", dst, err)
	}

	//nolint:gosec // statfs flags are a small non-negative bit mask
	kept := uintptr(stat.Flags) & inheriting
	flags := uintptr(unix.MS_REMOUNT|unix.MS_BIND|unix.MS_RDONLY) | kept

	if err := unix.Mount("", dst, "", flags, ""); err != nil {
		return fmt.Errorf("make %s read-only: %w", dst, err)
	}

	return nil
}

// makeTarget makes sure the mount point exists, creating it when possible.
func makeTarget(dst string, dir bool) error {
	//nolint:gosec // dst is under a private root this process is building
	if _, err := os.Stat(dst); err == nil {
		return nil
	}

	if err := mkdir(filepath.Dir(dst)); err != nil {
		return err
	}

	if dir {
		return mkdir(dst)
	}

	//nolint:gosec // dst is under a private root this process is building
	file, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, stubMode)
	if err != nil {
		if errors.Is(err, unix.EROFS) {
			return fmt.Errorf("%w: mount point %s is missing", ErrNoToolbox, dst)
		}

		return fmt.Errorf("create %s: %w", dst, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}

	return nil
}
