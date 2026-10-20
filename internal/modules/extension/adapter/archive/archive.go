// Package archive reads a project's build folder and keeps the zip of an extension the person downloads.
package archive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

const (
	maxFiles   = 3000
	maxTotal   = 50 << 20
	dirMode    = 0o750
	fileMode   = 0o640
	manifestIn = "manifest.json"
)

// Folders finds a project's folder.
type Folders interface {
	Dir(projectID string) (string, error)
}

// Packer implements application.Packer on the project folders and a folder of its own for the packages.
type Packer struct {
	folders Folders
	dir     string
}

// New keeps the packages in dir.
func New(folders Folders, dir string) *Packer { return &Packer{folders: folders, dir: dir} }

// open returns the build folder as a root that cannot reach outside it, even through a link.
func (p *Packer) open(projectID, folder string) (*os.Root, error) {
	if path.IsAbs(folder) || folder == ".." || strings.HasPrefix(folder, "../") {
		return nil, domain.ErrBadFolder
	}

	dir, err := p.folders.Dir(projectID)
	if err != nil {
		return nil, fmt.Errorf("find project folder: %w", err)
	}

	project, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open project folder: %w", err)
	}

	defer func() { _ = project.Close() }()

	build, err := project.OpenRoot(filepath.FromSlash(folder))
	if err != nil {
		return nil, domain.ErrBadFolder
	}

	return build, nil
}

// Inspect reads the folder: its manifest, which files it holds and how big it is.
func (p *Packer) Inspect(_ context.Context, projectID, folder string) (application.Tree, error) {
	root, err := p.open(projectID, folder)
	if err != nil {
		return application.Tree{}, err
	}

	defer func() { _ = root.Close() }()

	manifest, err := fs.ReadFile(root.FS(), manifestIn)
	if err != nil {
		return application.Tree{}, domain.ErrNoManifest
	}

	files, size, err := listFiles(root.FS())
	if err != nil {
		return application.Tree{}, err
	}

	return application.Tree{
		Manifest: manifest, Size: size, Files: len(files), Has: func(name string) bool { return files[path.Clean(name)] },
	}, nil
}

// listFiles names the regular files of the folder and adds up their size, stopping at the limits.
func listFiles(source fs.FS) (map[string]bool, int64, error) {
	files := map[string]bool{}

	var size int64

	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !entry.Type().IsRegular() {
			return walkErr
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("read %s: %w", name, infoErr)
		}

		files[name] = true
		size += info.Size()

		if len(files) > maxFiles || size > maxTotal {
			return domain.ErrTooLarge
		}

		return nil
	})
	if errors.Is(err, domain.ErrTooLarge) {
		return nil, 0, domain.ErrTooLarge
	}

	if err != nil {
		return nil, 0, fmt.Errorf("read folder: %w", err)
	}

	return files, size, nil
}

// Pack zips the folder into the project's package, replacing the last one.
func (p *Packer) Pack(_ context.Context, projectID, folder string) error {
	root, err := p.open(projectID, folder)
	if err != nil {
		return err
	}

	defer func() { _ = root.Close() }()

	if err = os.MkdirAll(p.dir, dirMode); err != nil {
		return fmt.Errorf("make package folder: %w", err)
	}

	temp, err := os.CreateTemp(p.dir, "pack-*.zip")
	if err != nil {
		return fmt.Errorf("make package: %w", err)
	}

	defer func() { _ = os.Remove(temp.Name()) }()

	if err = writeZip(temp, root.FS()); err != nil {
		_ = temp.Close()

		return err
	}

	if err = temp.Close(); err != nil {
		return fmt.Errorf("write package: %w", err)
	}

	if err = os.Chmod(temp.Name(), fileMode); err != nil {
		return fmt.Errorf("protect package: %w", err)
	}

	if err = os.Rename(temp.Name(), p.packagePath(projectID)); err != nil {
		return fmt.Errorf("keep package: %w", err)
	}

	return nil
}

// Open returns the project's package and its size.
func (p *Packer) Open(projectID string) (io.ReadCloser, int64, error) {
	file, err := os.Open(p.packagePath(projectID))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, domain.ErrNotFound
	}

	if err != nil {
		return nil, 0, fmt.Errorf("open package: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, 0, fmt.Errorf("read package: %w", err)
	}

	return file, info.Size(), nil
}

// Remove deletes the project's package; a missing one is not an error.
func (p *Packer) Remove(projectID string) error {
	if err := os.Remove(p.packagePath(projectID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove package: %w", err)
	}

	return nil
}

func (p *Packer) packagePath(projectID string) string {
	return filepath.Join(p.dir, filepath.Base(projectID)+".zip")
}
