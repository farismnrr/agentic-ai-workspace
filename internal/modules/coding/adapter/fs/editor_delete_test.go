package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestDeleteBinaryAndLargeFilesWithoutBackups(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	put(t, e.root, "clip.mp4", "video\x00bytes")
	large := make([]byte, domain.MaxWriteBytes+1)
	if err := os.WriteFile(filepath.Join(e.root, "large.txt"), large, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"clip.mp4", "large.txt"} {
		applied, err := e.editor.Delete(name)
		if err != nil {
			t.Fatalf("delete %s: %v", name, err)
		}
		if applied.Change.Op != "" || applied.Change.BackupID != "" {
			t.Errorf("delete %s created undo record %+v", name, applied.Change)
		}
		if _, err := os.Stat(filepath.Join(e.root, name)); !os.IsNotExist(err) {
			t.Errorf("%s was not removed", name)
		}
	}
}
