package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type counter struct{ n int }

func (c *counter) ID() (string, error) {
	c.n++

	return "abcdef0" + strconv.Itoa(c.n) + "00", nil
}

type env struct {
	editor *application.Editor
	reader *application.Service
	root   string
}

func newEnv(t *testing.T) env {
	t.Helper()

	root, outside := t.TempDir(), t.TempDir()
	put(t, root, "a.txt", "one two two\n")
	put(t, root, ".workspace/tasks.json", `{"tasks":{}}`)

	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	files, err := fs.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	backups, err := backup.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	return env{
		editor: application.NewEditor(files, files, backups, &counter{}),
		reader: application.NewService(files),
		root:   root,
	}
}

func (e env) content(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(e.root, name)) //nolint:gosec // reads the test's own temp workspace
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestEditNeedsAUniqueMatchUnlessAll(t *testing.T) {
	t.Parallel()

	e := newEnv(t)

	if _, err := e.editor.Edit("a.txt", "two", "2", false); !errors.Is(err, domain.ErrNotUnique) {
		t.Fatalf("ambiguous edit must be refused, got %v", err)
	}

	if _, err := e.editor.Edit("a.txt", "missing", "x", false); !errors.Is(err, domain.ErrNoMatch) {
		t.Fatalf("absent text must be refused, got %v", err)
	}

	if _, err := e.editor.Edit("a.txt", "two", "2", true); err != nil || e.content(t, "a.txt") != "one 2 2\n" {
		t.Fatalf("replace_all failed: %v %q", err, e.content(t, "a.txt"))
	}
}

func TestWriteCreatesParentsWithoutLeavingTempFiles(t *testing.T) {
	t.Parallel()

	e := newEnv(t)

	if _, err := e.editor.Write("deep/er/file.go", "package x\n"); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(filepath.Join(e.root, "deep/er"))
	if len(entries) != 1 || e.content(t, "deep/er/file.go") != "package x\n" {
		t.Fatalf("exactly the file should exist, got %v", entries)
	}
}

func TestUndoRestoresEveryKindOfChange(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	original := e.content(t, "a.txt")

	edit, _ := e.editor.Edit("a.txt", "one", "1", false)
	created, _ := e.editor.Write("new.txt", "hello")
	moved, _ := e.editor.Move("a.txt", "dir/b.txt")
	deleted, _ := e.editor.Delete("dir/b.txt")

	// Reverse in the opposite order, as a person pressing Undo down the list would.
	for _, applied := range []application.Applied{deleted, moved, created, edit} {
		if err := e.editor.Undo(applied.Change); err != nil {
			t.Fatalf("undo %s: %v", applied.Change.Op, err)
		}
	}

	if _, err := os.Stat(filepath.Join(e.root, "new.txt")); !os.IsNotExist(err) {
		t.Error("undoing a created file must remove it")
	}

	if e.content(t, "a.txt") != original {
		t.Errorf("a.txt should be back to %q, got %q", original, e.content(t, "a.txt"))
	}
}

func TestUndoRefusesWhenTheFileChangedAgain(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	first, _ := e.editor.Edit("a.txt", "one", "1", false)

	if _, err := e.editor.Edit("a.txt", "1", "uno", false); err != nil {
		t.Fatal(err)
	}

	if err := e.editor.Undo(first.Change); !errors.Is(err, domain.ErrChanged) {
		t.Fatalf("stale undo must be refused, got %v", err)
	}
}

func TestMoveNeverOverwritesAndDeleteIsFilesOnly(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	put(t, e.root, "b.txt", "b")
	put(t, e.root, "sub/c.txt", "c")

	if _, err := e.editor.Move("a.txt", "b.txt"); !errors.Is(err, domain.ErrExists) {
		t.Fatalf("move onto an existing file must fail, got %v", err)
	}

	if _, err := e.editor.Delete("sub"); !errors.Is(err, domain.ErrIsDir) {
		t.Fatalf("folders cannot be deleted, got %v", err)
	}
}

func TestWritesCannotEscapeOrTouchTheTaskFolder(t *testing.T) {
	t.Parallel()

	e := newEnv(t)

	for _, path := range []string{"escape/pwned.txt", ".workspace/tasks.json", ".env", ".git/hooks/pre-commit"} {
		if _, err := e.editor.Write(path, "x"); err == nil {
			t.Errorf("writing %q must be refused", path)
		}
	}
}

// A path that climbs out with ".." is clamped inside the root instead of escaping.
func TestTraversalIsClampedInsideTheRoot(t *testing.T) {
	t.Parallel()

	e := newEnv(t)

	if _, err := e.editor.Write("../../clamped.txt", "x"); err != nil {
		t.Fatal(err)
	}

	if e.content(t, "clamped.txt") != "x" {
		t.Fatal("the file should land at the workspace root")
	}
}
