package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// These site-owned runtime directories are the only workspace paths a site may write.
var writableWorkspacePaths = []string{"data/preview-runs", ".atja-runtime"}

// WorkspaceMounts validates the project root before enabling its approved writable paths.
func WorkspaceMounts(project string) ([]string, error) {
	if !filepath.IsAbs(project) || filepath.Clean(project) != project {
		return nil, errors.New("cannot enable Sites writable paths; ask the project owner to check the workspace")
	}
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil || resolved != project {
		return nil, errors.New("cannot enable Sites writable paths; ask the project owner to check the workspace")
	}
	projectInfo, err := os.Lstat(project)
	if err != nil || !projectInfo.IsDir() || projectInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("cannot enable Sites writable paths; ask the project owner to check the workspace")
	}
	projectRoot, err := os.OpenRoot(project)
	if err != nil {
		return nil, errors.New("cannot enable Sites writable paths; ask the project owner to check the workspace")
	}
	defer projectRoot.Close()
	openedRoot, err := projectRoot.Stat(".")
	if err != nil || !os.SameFile(projectInfo, openedRoot) {
		return nil, errors.New("cannot enable Sites writable paths; ask the project owner to check the workspace")
	}

	mounts := make([]string, 0, len(writableWorkspacePaths))
	for _, rel := range writableWorkspacePaths {
		if err := ensureProjectDir(projectRoot, rel); err != nil {
			return nil, fmt.Errorf("cannot enable writable path %q: %w", rel, err)
		}
		mounts = append(mounts, rel)
	}

	return mounts, nil
}

// EnsureRuntimeStorage creates the per-site storage root and data directory safely.
func EnsureRuntimeStorage(base, dir string) error {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base {
		return errors.New("invalid Sites storage directory")
	}
	parent, name := filepath.Dir(base), filepath.Base(base)
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("Sites storage parent is unavailable")
	}
	parentRoot, err := os.OpenRoot(parent)
	if err != nil {
		return errors.New("Sites storage parent cannot be opened safely")
	}
	defer parentRoot.Close()
	if err = parentRoot.Mkdir(name, dirMode); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("Sites storage directory cannot be created")
	}
	info, err := parentRoot.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Sites storage directory is unsafe")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return errors.New("Sites storage cannot be opened safely")
	}
	defer root.Close()

	rel, err := filepath.Rel(base, dir)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || !filepath.IsLocal(rel) {
		return errors.New("invalid Sites storage directory")
	}
	if err := ensureRuntimeDir(root, rel); err != nil {
		return err
	}
	return ensureRuntimeDir(root, filepath.Join(rel, "data"))
}

func ensureRuntimeDir(root *os.Root, rel string) error {
	if rel == "." || !filepath.IsLocal(rel) {
		return errors.New("invalid Sites storage path")
	}
	current := ""
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("invalid Sites storage path")
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err = root.Mkdir(current, dirMode); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("Sites storage contains an unsafe path")
		}
	}
	return nil
}

func ensureProjectDir(root *os.Root, rel string) error {
	if !filepath.IsLocal(rel) || filepath.Clean(rel) != rel || strings.Contains(rel, `\`) {
		return errors.New("path is not an approved relative directory")
	}
	current := ""
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return errors.New("path is not an approved relative directory")
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err = root.Mkdir(current, dirMode); err != nil && !errors.Is(err, os.ErrExist) {
				return errors.New("directory could not be created; ask the administrator to check workspace storage")
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return errors.New("directory could not be checked; ask the administrator to check workspace storage")
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("path contains a file or symlink; the project owner must replace it with a directory")
		}
	}
	return nil
}
