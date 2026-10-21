package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path"
	"syscall"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Root exposes a directory read-only. It is built on os.Root, so the standard
// library rejects ".." and symlinks that would leave the directory.
type Root struct {
	root *os.Root
	fsys iofs.FS
}

// Open confines the workspace to dir.
func Open(dir string) (*Root, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open workspace root: %w", err)
	}

	return &Root{root: root, fsys: root.FS()}, nil
}

// ReadDir lists a directory.
func (r *Root) ReadDir(rel string) ([]domain.Entry, error) {
	items, err := iofs.ReadDir(r.fsys, rel)
	if err != nil {
		return nil, translate(err)
	}

	entries := make([]domain.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, entryOf(path.Join(rel, item.Name()), item))
	}

	return entries, nil
}

// ReadFile reads up to max bytes of a regular file and reports whether more existed.
func (r *Root) ReadFile(rel string, max int64) ([]byte, bool, error) {
	data, err := r.readCapped(rel, max+1)
	if err != nil {
		return nil, false, err
	}

	if int64(len(data)) > max {
		return data[:max], true, nil
	}

	return data, false, nil
}

// Open returns a seekable file confined to the workspace and its size.
func (r *Root) Open(rel string) (io.ReadSeekCloser, int64, error) {
	file, err := r.root.Open(rel)
	if err != nil {
		return nil, 0, translate(err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, translate(err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, 0, domain.ErrIsDir
	}

	return file, info.Size(), nil
}

func (r *Root) readCapped(rel string, limit int64) ([]byte, error) {
	file, err := r.fsys.Open(rel)
	if err != nil {
		return nil, translate(err)
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, translate(err)
	}

	if info.IsDir() {
		return nil, domain.ErrIsDir
	}

	data, err := io.ReadAll(io.LimitReader(file, limit))

	return data, translate(err)
}

// Walk visits every entry below rel.
func (r *Root) Walk(rel string, visit func(entry domain.Entry) error) error {
	err := iofs.WalkDir(r.fsys, rel, func(p string, item iofs.DirEntry, walkErr error) error {
		if walkErr != nil || p == rel {
			return nil //nolint:nilerr // unreadable entries are skipped, not fatal
		}

		return skipDir(visit(entryOf(p, item)))
	})

	return translate(err)
}

func entryOf(rel string, item iofs.DirEntry) domain.Entry {
	entry := domain.Entry{Path: rel, Dir: item.IsDir()}
	if info, err := item.Info(); err == nil && !entry.Dir {
		entry.Size = info.Size()
	}

	return entry
}

func skipDir(err error) error {
	if errors.Is(err, application.ErrSkipDir) {
		return iofs.SkipDir
	}

	return err
}

var knownErrors = []struct{ from, to error }{
	{syscall.ENOTDIR, domain.ErrNotDir},
	{iofs.ErrNotExist, domain.ErrNotFound},
	{iofs.ErrInvalid, domain.ErrDenied},
	{iofs.ErrPermission, domain.ErrDenied},
}

// translate maps io errors to the domain's vocabulary; anything unexpected about
// a path (such as a symlink leaving the root) is reported as denied.
func translate(err error) error {
	if err == nil {
		return nil
	}

	for _, known := range knownErrors {
		if errors.Is(err, known.from) {
			return known.to
		}
	}

	var pathErr *iofs.PathError
	if errors.As(err, &pathErr) {
		return domain.ErrDenied
	}

	return fmt.Errorf("workspace io: %w", err)
}
