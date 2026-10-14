package bootstrap

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	mydataapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/application"
)

// writeFiles copies a project's regular files; links and folders themselves are skipped.
// The folder is read through os.Root, so a link swapped in while the export runs
// can never lead the copy outside the project.
func (p projectsExport) writeFiles(projectID, base string, out mydataapp.Archive) error {
	dir, err := p.folders.Dir(projectID)
	if err != nil {
		return fmt.Errorf("project folder: %w", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open project folder: %w", err)
	}

	defer func() { _ = root.Close() }()

	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || !entry.Type().IsRegular() {
			return walkErr
		}

		return copyFile(out, base+"/"+path, root, path)
	})
	if err != nil {
		return fmt.Errorf("copy project files: %w", err)
	}

	return nil
}

func copyFile(out mydataapp.Archive, name string, root *os.Root, path string) error {
	src, err := root.Open(path)
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
