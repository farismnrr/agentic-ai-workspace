package fs

import (
	"errors"
	iofs "io/fs"
	"os"
	"path"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	dirMode     = 0o755
	newFileMode = 0o644
	tasksFile   = ".workspace/tasks.json"
)

// Stat describes one path inside the workspace.
func (r *Root) Stat(rel string) (domain.Entry, error) {
	info, err := r.root.Stat(rel)
	if err != nil {
		return domain.Entry{}, translate(err)
	}

	return domain.Entry{Path: rel, Dir: info.IsDir(), Size: info.Size()}, nil
}

// WriteFile replaces or creates a file atomically: the data goes to a sibling
// temporary file that is then renamed over the target.
func (r *Root) WriteFile(rel string, data []byte) error {
	mode := os.FileMode(newFileMode)
	if info, err := r.root.Stat(rel); err == nil {
		mode = info.Mode().Perm()
	}

	if err := r.root.MkdirAll(path.Dir(rel), dirMode); err != nil {
		return translate(err)
	}

	temp := path.Join(path.Dir(rel), "."+path.Base(rel)+".writing")
	if err := r.root.WriteFile(temp, data, mode); err != nil {
		return translate(err)
	}

	if err := r.root.Rename(temp, rel); err != nil {
		_ = r.root.Remove(temp)

		return translate(err)
	}

	return nil
}

// Remove deletes one file.
func (r *Root) Remove(rel string) error {
	return translate(r.root.Remove(rel))
}

// Rename moves a file, creating the destination's folders.
func (r *Root) Rename(from, to string) error {
	if err := r.root.MkdirAll(path.Dir(to), dirMode); err != nil {
		return translate(err)
	}

	return translate(r.root.Rename(from, to))
}

// ReadTasks returns the raw task definitions. This is a server-side read: the
// tools themselves can never reach the .workspace folder.
func (r *Root) ReadTasks() ([]byte, error) {
	data, err := r.root.ReadFile(tasksFile)
	if errors.Is(err, iofs.ErrNotExist) {
		return nil, domain.ErrNoTasks
	}

	return data, translate(err)
}
