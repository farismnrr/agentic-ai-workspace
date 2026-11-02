package site_test

import (
	"os"
	"path/filepath"
	"testing"

	siteruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/runtime"
)

func TestSitesStorageCreatesMissingBaseAndPreservesData(t *testing.T) {
	dataDir := t.TempDir()
	base := filepath.Join(dataDir, "sites")
	siteDir := filepath.Join(base, "project")

	if err := siteruntime.EnsureRuntimeStorage(base, siteDir); err != nil {
		t.Fatalf("create missing Sites storage: %v", err)
	}

	database := filepath.Join(siteDir, "data", "app.sqlite")
	if err := os.WriteFile(database, []byte("persisted"), 0o600); err != nil {
		t.Fatalf("write runtime data: %v", err)
	}
	if err := siteruntime.EnsureRuntimeStorage(base, siteDir); err != nil {
		t.Fatalf("reuse Sites storage: %v", err)
	}
	content, err := os.ReadFile(database)
	if err != nil || string(content) != "persisted" {
		t.Fatalf("runtime data was not preserved: content=%q err=%v", content, err)
	}
}

func TestSitesStorageRejectsSymlinkBase(t *testing.T) {
	dataDir := t.TempDir()
	target := filepath.Join(dataDir, "outside")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dataDir, "sites")
	if err := os.Symlink(target, base); err != nil {
		t.Fatal(err)
	}

	if err := siteruntime.EnsureRuntimeStorage(base, filepath.Join(base, "project")); err == nil {
		t.Fatal("expected symlink Sites base to be rejected")
	}
	if _, err := os.Stat(filepath.Join(target, "project")); !os.IsNotExist(err) {
		t.Fatalf("unsafe target was modified: %v", err)
	}
}

func TestSitesWritablePathsRejectSymlinkedProjectAncestor(t *testing.T) {
	root := t.TempDir()
	storage := filepath.Join(root, "storage")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(filepath.Join(storage, "workspaces"), 0o700); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(outside, "project")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(storage, "workspaces", "redirect")); err != nil {
		t.Fatal(err)
	}

	lexicalProject := filepath.Join(storage, "workspaces", "redirect", "project")
	if _, err := siteruntime.WorkspaceMounts(lexicalProject); err == nil {
		t.Fatal("expected symlinked project ancestor to be rejected")
	}
	if _, err := os.Stat(filepath.Join(project, "data", "preview-runs")); !os.IsNotExist(err) {
		t.Fatalf("rejected path was modified: %v", err)
	}
}
