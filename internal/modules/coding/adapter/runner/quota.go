package runner

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// DefaultWorkspaceBytes is how large a workspace may grow while a task runs.
	DefaultWorkspaceBytes = 2 << 30
	quotaCheckEvery       = time.Second
)

// watchSize polls the workspace while a task runs and calls stop when it grows past
// limit. A filesystem quota would be tighter but needs privileges the server lacks.
// The result turns true once the limit was hit.
func watchSize(ctx context.Context, dir string, limit int64, stop func()) *atomic.Bool {
	exceeded := &atomic.Bool{}
	if limit <= 0 {
		return exceeded
	}

	go func() {
		ticker := time.NewTicker(quotaCheckEvery)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if sizeOver(dir, limit) {
					exceeded.Store(true)
					stop()

					return
				}
			}
		}
	}()

	return exceeded
}

// sizeOver reports whether the regular files under dir add up to more than limit,
// stopping the walk as soon as they do. Transient build and preview run outputs
// are excluded so they do not block routine project and Git commands.
func sizeOver(dir string, limit int64) bool {
	var total int64

	errOver := fs.ErrClosed // any sentinel that ends the walk early

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			rel, relErr := filepath.Rel(dir, path)
			if relErr == nil && rel != "." {
				base := filepath.Base(path)
				if strings.HasPrefix(base, ".build-output-") || strings.HasPrefix(base, ".artifact-output-") ||
					rel == "data/preview-runs" || rel == ".atja-runtime" || strings.HasPrefix(base, ".playwright") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil // files that vanish while walking do not count
		}

		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}

		if total > limit {
			return errOver
		}

		return nil
	})

	return err != nil && total > limit
}
