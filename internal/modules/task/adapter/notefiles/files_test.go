package notefiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/notefiles"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
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

func TestRemoveTaskDeletesOnlyAValidatedTaskFolder(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	store := notefiles.New(func(string) (string, error) { return project, nil })
	root := filepath.Join(project, "project-notes", "tasks")
	mustMakeDir(t, filepath.Join(root, "000000000000000000000001"), 0o750)
	mustWriteFile(t, filepath.Join(root, "000000000000000000000001", "plan.md"), "task", 0o600)
	shared := filepath.Join(project, "uploads")
	mustMakeDir(t, shared, 0o750)
	mustWriteFile(t, filepath.Join(shared, "keep.txt"), "shared", 0o600)
	if err := store.RemoveTask(t.Context(), "project", "../../uploads"); err == nil {
		t.Fatal("traversal-shaped task id was accepted")
	}
	if _, err := os.Stat(filepath.Join(shared, "keep.txt")); err != nil {
		t.Fatalf("shared upload was touched: %v", err)
	}
	if err := store.RemoveTask(t.Context(), "project", "000000000000000000000001"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "000000000000000000000001")); !os.IsNotExist(err) {
		t.Fatalf("task notes remain: %v", err)
	}
}

func TestRemoveTaskDeletesFolderForGeneratedTaskID(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	taskID, err := (opaque.Generator{}).ID()
	if err != nil {
		t.Fatal(err)
	}
	if len(taskID) != 32 {
		t.Fatalf("generated task id has length %d, want 32", len(taskID))
	}
	taskFolder := filepath.Join(project, "project-notes", "tasks", taskID)
	mustMakeDir(t, taskFolder, 0o750)
	mustWriteFile(t, filepath.Join(taskFolder, "plan.md"), "task", 0o600)
	store := notefiles.New(func(string) (string, error) { return project, nil })
	if err := store.RemoveTask(t.Context(), "project", taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(taskFolder); !os.IsNotExist(err) {
		t.Fatalf("generated task notes remain: %v", err)
	}
}

func mustMakeDir(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, mode); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
