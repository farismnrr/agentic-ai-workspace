package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
)

func TestStreamCreateRefusesOverwriteAndSymlinkEscape(t *testing.T) {
	t.Parallel()
	rootDir, outside := t.TempDir(), t.TempDir()
	root, err := fs.Open(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Create("uploads/a.txt", strings.NewReader("first")); err != nil {
		t.Fatal(err)
	}
	if err := root.Create("uploads/a.txt", strings.NewReader("second")); err == nil {
		t.Fatal("Create overwrote an existing file")
	}
	if err := os.Symlink(outside, filepath.Join(rootDir, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := root.Create("escape/pwned.txt", strings.NewReader("no")); err == nil {
		t.Fatal("Create followed a symlink outside the workspace")
	}
	if _, err := os.Stat(filepath.Join(outside, "pwned.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside file exists: %v", err)
	}
}

type brokenReader struct{ sent bool }

var errReadFailed = errors.New("read failed")

func (r *brokenReader) Read(data []byte) (int, error) {
	if r.sent {
		return 0, errReadFailed
	}
	r.sent = true
	return copy(data, []byte("partial")), nil
}

func TestStreamCreateRemovesPartialFileOnReadFailure(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	root, err := fs.Open(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Create("uploads/partial.dat", &brokenReader{}); err == nil {
		t.Fatal("Create accepted a failing reader")
	}
	if _, err := os.Stat(filepath.Join(rootDir, "uploads", "partial.dat")); !os.IsNotExist(err) {
		t.Fatalf("partial file remains: %v", err)
	}
}
