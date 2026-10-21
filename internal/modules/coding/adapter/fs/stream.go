package fs

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
)

// Create streams a new file into the workspace and removes partial data on failure.
func (r *Root) Create(rel string, src io.Reader) error {
	if err := r.root.MkdirAll(path.Dir(rel), dirMode); err != nil {
		return translate(err)
	}
	file, err := r.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
	if err != nil {
		return translate(err)
	}
	if _, err = io.Copy(file, src); err != nil {
		_ = file.Close()
		_ = r.root.Remove(rel)
		return fmt.Errorf("stream workspace file: %w", err)
	}
	if err = file.Close(); err != nil {
		_ = r.root.Remove(rel)
		return fmt.Errorf("close workspace file: %w", err)
	}
	return nil
}

var _ application.Writer = (*Root)(nil)
