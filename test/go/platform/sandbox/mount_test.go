package sandbox_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
	"github.com/moby/moby/api/types/mount"
)

func TestMountsRejectSymlinkedWorkspaceAncestor(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(filepath.Join(dataDir, "workspaces"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "project"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dataDir, "workspaces", "redirect")); err != nil {
		t.Fatal(err)
	}

	for _, volume := range []string{"", "workspace-volume"} {
		cfg := sandbox.Config{DataDir: dataDir, DataVolume: volume}
		_, err := cfg.Mounts(sandbox.Spec{
			Workspace: filepath.Join(dataDir, "workspaces", "redirect", "project"),
		})
		if err == nil {
			t.Fatalf("expected symlink ancestor to be rejected for volume %q", volume)
		}
	}
}

func TestMountsKeepWorkspaceReadonlyAndWritableSubpathsNarrow(t *testing.T) {
	dataDir := t.TempDir()
	workspace := filepath.Join(dataDir, "projects", "project")
	previewRuns := filepath.Join(workspace, "data", "preview-runs")
	for _, dir := range []string{previewRuns, filepath.Join(workspace, ".atja-runtime")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	existing := filepath.Join(previewRuns, "existing.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, volume := range []string{"", "workspace-volume"} {
		cfg := sandbox.Config{DataDir: dataDir, DataVolume: volume}
		got, err := cfg.Mounts(sandbox.Spec{
			Workspace: workspace, WorkspaceReadOnly: true,
			WorkspaceWritable: []string{"data/preview-runs", ".atja-runtime"},
		})
		if err != nil {
			t.Fatalf("build mounts for volume %q: %v", volume, err)
		}
		if len(got) != 3 || got[0].Target != sandbox.Workspace || !got[0].ReadOnly {
			t.Fatalf("workspace mount is not read-only or mount count changed: %+v", got)
		}
		if got[1].Target != filepath.Join(sandbox.Workspace, "data", "preview-runs") || got[1].ReadOnly ||
			got[2].Target != filepath.Join(sandbox.Workspace, ".atja-runtime") || got[2].ReadOnly {
			t.Fatalf("writable mounts escaped the approved paths: %+v", got)
		}
		if volume == "" {
			if got[1].Type != mount.TypeBind || got[1].Source != previewRuns {
				t.Fatalf("bind source changed: %+v", got[1])
			}
			continue
		}
		if got[1].Type != mount.TypeVolume || got[1].Source != volume || got[1].VolumeOptions.Subpath != "projects/project/data/preview-runs" {
			t.Fatalf("volume subpath changed: %+v", got[1])
		}
	}
	content, err := os.ReadFile(existing)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing project data was changed: content=%q err=%v", content, err)
	}
}
