package bootstrap

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	mydataapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/application"
)

// writeFiles copies a project's regular files; links and folders themselves are skipped.
func (p projectsExport) writeFiles(projectID, base string, out mydataapp.Archive) error {
	dir, err := p.folders.Dir(projectID)
	if err != nil {
		return fmt.Errorf("project folder: %w", err)
	}

	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || !entry.Type().IsRegular() {
			return walkErr
		}

		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return fmt.Errorf("relative path: %w", relErr)
		}

		return copyFile(out, base+"/"+filepath.ToSlash(rel), path)
	})
	if err != nil {
		return fmt.Errorf("copy project files: %w", err)
	}

	return nil
}

func copyFile(out mydataapp.Archive, name, path string) error {
	src, err := os.Open(path) //nolint:gosec // path comes from walking a folder the server created
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}

	defer func() { _ = src.Close() }()

	dst, err := out.Create(name)
	if err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy %s: %w", name, err)
	}

	return nil
}
