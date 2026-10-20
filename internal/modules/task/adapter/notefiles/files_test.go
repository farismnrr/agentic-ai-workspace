package notefiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/notefiles"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestDocumentExportsCannotFollowASymlinkOutsideTheProject(t *testing.T) {
	t.Parallel()
	project, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, "project-notes")); err != nil {
		t.Fatal(err)
	}
	store := notefiles.New(func(string) (string, error) { return project, nil })
	files := []application.NoteFile{{Path: "project-notes/input.txt", Content: "private"}}
	err := store.Write(t.Context(), "project", files)
	if err == nil {
		t.Fatal("an export followed a symlink out of the project")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside directory was touched: %v %v", entries, err)
	}
}
