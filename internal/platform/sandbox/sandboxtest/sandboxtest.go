// Package sandboxtest gives tests a real toolbox without touching the host's programs.
// The toolbox is just Alpine's minimal root filesystem (busybox), built once and cached.
package sandboxtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const (
	installTimeout = 5 * time.Minute
	dirPerm        = 0o750
	filePerm       = 0o600
)

// Ensure returns a sandbox folder (the parent of "general") with a ready toolbox. The
// AAW_TEST_SANDBOX variable points at an existing one; otherwise a base toolbox is built
// into the user's cache the first time. Tests that need it skip when it cannot be had.
func Ensure() (string, error) {
	if dir := os.Getenv("AAW_TEST_SANDBOX"); dir != "" {
		return dir, ready(dir)
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find a cache folder: %w", err)
	}

	dir := filepath.Join(cache, "agentic-ai-workspace", "test-sandbox")
	if err = os.MkdirAll(dir, dirPerm); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	// Test binaries of different packages run in parallel: one builds, the others wait.
	lock, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_RDWR, filePerm) //nolint:gosec // our own cache
	if err != nil {
		return "", fmt.Errorf("open lock: %w", err)
	}

	defer func() { _ = lock.Close() }()

	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX); err != nil {
		return "", fmt.Errorf("lock %s: %w", dir, err)
	}

	if ready(dir) == nil {
		return dir, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	if err := sandbox.Install(ctx, sandbox.InstallOptions{Dir: filepath.Join(dir, "general")}); err != nil {
		return "", fmt.Errorf("build the test toolbox: %w", err)
	}

	return dir, ready(dir)
}

func ready(dir string) error {
	if err := sandbox.ToolboxReady(filepath.Join(dir, "general")); err != nil {
		return errors.Join(errors.New("test toolbox is not ready"), err) //nolint:err113 // a test helper's message
	}

	return nil
}
