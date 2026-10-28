package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"syscall"
	"time"
)

const (
	maxEffectEntries = 4096
	maxEffectBytes   = 64 << 20
	maxEffectTime    = 2 * time.Second
)

// Snapshot deadlines are cooperative: they stop traversal and chunked reads,
// but cannot interrupt a blocked filesystem metadata call or close operation.

var (
	errEffectSnapshotLimit = errors.New("snapshot limit reached")
	errGitMetadataOutside  = errors.New("Git metadata is outside the project")
	errSymlinkOutsideRoot  = errors.New("snapshot symlink target is outside the project")
	errSnapshotChanged     = errors.New("snapshot entry changed")
	errSnapshotSymlink     = errors.New("snapshot symlink changed")
)

type effectEntry struct {
	info   os.FileInfo
	digest string
}

type effectSnapshot struct {
	files    map[string]effectEntry
	complete bool
}

func snapshotEffects(ctx context.Context, root string) (snapshot effectSnapshot) {
	scanCtx, cancel := context.WithTimeout(ctx, maxEffectTime)
	defer cancel()

	workspace, err := os.OpenRoot(root)
	if err != nil {
		return effectSnapshot{}
	}

	scanner := effectScanner{state: &effectScanState{root: workspace, snapshot: effectSnapshot{
		files: map[string]effectEntry{}, complete: true,
	}}}
	defer func() {
		if closeErr := workspace.Close(); closeErr != nil {
			snapshot.complete = false
		}
	}()
	err = fs.WalkDir(workspace.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if scanErr := scanCtx.Err(); scanErr != nil {
			return fmt.Errorf("stop effect scan: %w", scanErr)
		}
		return scanner.visit(scanCtx, name, entry, walkErr)
	})
	if err == nil && scanCtx.Err() == nil {
		err = scanner.verifyEntries(scanCtx)
	}
	if err != nil || scanCtx.Err() != nil {
		scanner.state.snapshot.complete = false
	}
	if scanner.state.counters.externalGitConfig {
		scanner.state.snapshot.complete = false
	}
	return scanner.state.snapshot
}

type effectScanner struct {
	state *effectScanState
}

type effectScanState struct {
	root     *os.Root
	snapshot effectSnapshot
	counters effectScanCounters
}

type effectScanCounters struct {
	bytes             int64
	externalGitConfig bool
}

func (s *effectScanner) visit(ctx context.Context, name string, entry fs.DirEntry, walkErr error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop effect scan: %w", err)
	}
	if walkErr != nil {
		return fmt.Errorf("walk project snapshot: %w", walkErr)
	}
	name = path.Clean(name)
	if name == "." {
		return s.captureRoot()
	}
	if skip, err := skipInertGitPath(name, entry); skip || err != nil {
		return err
	}
	return s.captureEntry(ctx, name, entry)
}

func (s *effectScanner) captureRoot() error {
	info, err := s.state.root.Stat(".")
	if err != nil {
		return fmt.Errorf("stat workspace root: %w", err)
	}
	s.state.snapshot.files["."] = effectEntry{info: info}
	return nil
}

func (s *effectScanner) captureEntry(ctx context.Context, name string, entry fs.DirEntry) error {
	if len(s.state.snapshot.files) >= maxEffectEntries {
		return errEffectSnapshotLimit
	}
	walkedInfo, err := entry.Info()
	if err != nil {
		return fmt.Errorf("read snapshot entry info: %w", err)
	}
	info, err := s.state.root.Lstat(name)
	if err != nil || !sameMetadata(walkedInfo, info) {
		return fmt.Errorf("%w: %s", errSnapshotChanged, name)
	}
	if name == ".git" && !info.IsDir() {
		return fmt.Errorf("%w: Git directory pointer", errGitMetadataOutside)
	}
	if info.IsDir() {
		s.state.snapshot.files[name] = effectEntry{info: info}
		return nil
	}
	digest, err := s.entryDigest(ctx, name, info)
	if err != nil {
		return err
	}
	s.state.snapshot.files[name] = effectEntry{info: info, digest: digest}
	return nil
}

func (s *effectScanner) entryDigest(ctx context.Context, name string, info os.FileInfo) (string, error) {
	if info.Mode().IsRegular() {
		return s.digest(ctx, name, info)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return "", nil
	}
	target, err := s.state.root.Readlink(name)
	if err != nil {
		return "", fmt.Errorf("read snapshot symlink: %w", err)
	}
	if confinedErr := confinedSymlinkTarget(s.state.root, name); confinedErr != nil {
		return "", confinedErr
	}
	after, err := s.state.root.Lstat(name)
	if err != nil || !sameMetadata(info, after) {
		return "", fmt.Errorf("%w: %s", errSnapshotSymlink, name)
	}
	hash := sha256.Sum256([]byte(target))
	return hex.EncodeToString(hash[:]), nil
}

func confinedSymlinkTarget(root *os.Root, name string) error {
	target, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("%w: %s", errSymlinkOutsideRoot, name)
	}
	info, statErr := target.Stat()
	if statErr != nil {
		_ = target.Close()
		return fmt.Errorf("stat snapshot symlink target: %w", statErr)
	}
	if !info.Mode().IsRegular() && !info.IsDir() {
		_ = target.Close()
		return fmt.Errorf("%w: non-regular target %s", errSnapshotSymlink, name)
	}
	if closeErr := target.Close(); closeErr != nil {
		return fmt.Errorf("close symlink target: %w", closeErr)
	}
	return nil
}

func (s *effectScanner) verifyEntries(ctx context.Context) error {
	for name, entry := range s.state.snapshot.files {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("verify snapshot stability: %w", err)
		}
		after, err := s.state.root.Lstat(name)
		if err != nil || !sameMetadata(entry.info, after) {
			return fmt.Errorf("%w: %s", errSnapshotChanged, name)
		}
	}
	return nil
}
