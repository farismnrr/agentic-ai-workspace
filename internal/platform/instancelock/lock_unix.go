//go:build unix

// Package instancelock lets one process own a file for as long as it runs, so a feature that keeps state in memory
// can refuse to start twice against the same data folder.
package instancelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const fileMode = 0o600

// ErrHeld means another process already holds the lock.
var ErrHeld = errors.New("another instance already holds the lock")

// Acquire takes an exclusive lock on the file at path, creating it if needed, without waiting. The lock lasts until
// the returned function is called or the process ends, whichever is first. flock is advisory and only reliable
// between processes on the same host, which is the supported setup.
func Acquire(path string) (release func(), err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, fileMode) //nolint:gosec // path is built by the server
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()

		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrHeld
		}

		return nil, fmt.Errorf("lock %s: %w", path, err)
	}

	return func() { _ = file.Close() }, nil
}
