package sandbox_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// This uses the actual volume Subpath mount builder used by a deployed app.
// Run it against a disposable volume prepopulated by the volume isolation scenario.
func TestOwnerVolumeSubpathsAreIsolated(t *testing.T) {
	volume, image := os.Getenv("VOLUME_ISOLATION_VOLUME"), os.Getenv("VOLUME_ISOLATION_IMAGE")
	if volume == "" || image == "" {
		t.Skip("VOLUME_ISOLATION_VOLUME and VOLUME_ISOLATION_IMAGE required for Docker integration")
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	outsideDir := filepath.Join(filepath.Dir(dataDir), "outside")
	prepareVolumeMountSources(t, dataDir, outsideDir)

	cfg := sandbox.Config{
		Image: image, DataVolume: volume, DataDir: dataDir,
		UID: 65532, GID: 65532, Limits: sandbox.DefaultLimits(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine, err := sandbox.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Docker must be available for volume isolation: %v", err)
	}

	runOwnerScope(t, ctx, engine, dataDir, "a", "b")
	runOwnerScope(t, ctx, engine, dataDir, "b", "a")
	runSiteWritableVolume(t, ctx, engine, dataDir, "a", "b")
	runSiteWritableVolume(t, ctx, engine, dataDir, "b", "a")
	runReadonlyBuildOutput(t, ctx, engine, dataDir, "a")
	runReadonlyBuildOutput(t, ctx, engine, dataDir, "b")
	assertOutsideDataRejected(t, ctx, engine, outsideDir)
}

func prepareVolumeMountSources(t *testing.T, dataDir, outsideDir string) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(dataDir, "workspaces", "a", "data", "preview-runs"),
		filepath.Join(dataDir, "workspaces", "a", ".atja-runtime"),
		filepath.Join(dataDir, "workspaces", "a", "builds"),
		filepath.Join(dataDir, "workspaces", "b", "data", "preview-runs"),
		filepath.Join(dataDir, "workspaces", "b", ".atja-runtime"),
		filepath.Join(dataDir, "workspaces", "b", "builds"),
		filepath.Join(dataDir, "caches", "a"),
		filepath.Join(dataDir, "caches", "b"),
		outsideDir,
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("create temporary mount source %q: %v", path, err)
		}
	}
}

func runSiteWritableVolume(t *testing.T, ctx context.Context, engine *sandbox.Docker, dataDir, scope, other string) {
	t.Helper()
	spec := sandbox.Spec{
		Workspace: volumeWorkspace(dataDir, scope), WorkspaceReadOnly: true,
		WorkspaceWritable: []string{"data/preview-runs", ".atja-runtime"},
		Argv: []string{"sh", "-c", "test \"$(cat /workspace/data/preview-runs/existing.txt)\" = before-" + scope + " && " +
			"printf updated >> /workspace/data/preview-runs/existing.txt && " +
			"touch /workspace/.atja-runtime/start && " +
			"test ! -e /workspace/marker-" + other + " && " +
			"if touch /workspace/read-only-probe 2>/dev/null; then exit 42; fi && " +
			"test -e /workspace/.atja-runtime/start && echo SITE_VOLUME_MOUNTS_OK"},
	}
	var output bytes.Buffer
	result, err := engine.Run(ctx, spec, &output)
	if err != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "SITE_VOLUME_MOUNTS_OK") {
		t.Fatalf("Sites volume mounts for %s: code=%d err=%v output=%q", scope, result.ExitCode, err, output.String())
	}

	result, err = engine.Run(ctx, sandbox.Spec{
		Workspace: volumeWorkspace(dataDir, scope),
		Argv:      []string{"sh", "-c", "test \"$(cat /workspace/data/preview-runs/existing.txt)\" = before-" + scope + "updated"},
	}, io.Discard)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("Sites volume data was not retained for %s: code=%d err=%v", scope, result.ExitCode, err)
	}
}

func TestSiteWritableDirectoriesUseTheOriginalBindWorkspace(t *testing.T) {
	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	dataDir := t.TempDir()
	workspace := filepath.Join(dataDir, "projects", "deadbeef")
	previewRuns := filepath.Join(workspace, "data", "preview-runs")
	if err := os.MkdirAll(previewRuns, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, ".atja-runtime"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(previewRuns, "existing.txt"), []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine, err := sandbox.Open(ctx, sandbox.Config{
		Image: image, DataDir: dataDir, UID: os.Getuid(), GID: os.Getgid(), Limits: sandbox.DefaultLimits(),
	})
	if err != nil {
		t.Fatalf("Docker must be available for Sites bind mount coverage: %v", err)
	}

	var output bytes.Buffer
	result, err := engine.Run(ctx, sandbox.Spec{
		Workspace: workspace, WorkspaceReadOnly: true,
		WorkspaceWritable: []string{"data/preview-runs", ".atja-runtime"},
		Argv: []string{"sh", "-c", "test \"$(cat /workspace/data/preview-runs/existing.txt)\" = kept && " +
			"printf updated >> /workspace/data/preview-runs/existing.txt && " +
			"mkdir -p /workspace/.atja-runtime && touch /workspace/.atja-runtime/start && " +
			"if touch /workspace/read-only-probe 2>/dev/null; then exit 41; fi && " +
			"test \"$(cat /workspace/data/preview-runs/existing.txt)\" = keptupdated && " +
			"test -e /workspace/.atja-runtime/start && echo SITE_BIND_MOUNTS_OK"},
	}, &output)
	if err != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "SITE_BIND_MOUNTS_OK") {
		t.Fatalf("Sites writable bind mounts: code=%d err=%v output=%q", result.ExitCode, err, output.String())
	}
	content, err := os.ReadFile(filepath.Join(previewRuns, "existing.txt"))
	if err != nil || string(content) != "keptupdated" {
		t.Fatalf("project file was not updated in place: content=%q err=%v", content, err)
	}
}

func runOwnerScope(t *testing.T, ctx context.Context, engine *sandbox.Docker, dataDir, scope, other string) {
	t.Helper()
	spec := sandbox.Spec{
		Workspace: volumeWorkspace(dataDir, scope),
		Cache:     filepath.Join(dataDir, "caches", scope),
		Argv: []string{
			"sh", "-c",
			"test -e /workspace/marker-" + scope +
				" && test ! -e /workspace/marker-" + other +
				" && test -e /cache/cache-" + scope +
				" && test ! -e /cache/cache-" + other +
				" && touch /workspace/wrote-" + scope +
				" && mkdir -p /workspace/builds" +
				" && echo SUBPATH_OK",
		},
	}
	var output bytes.Buffer
	result, err := engine.Run(ctx, spec, &output)
	if err != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "SUBPATH_OK") {
		t.Fatalf("scope %s: code=%d err=%v output=%q", scope, result.ExitCode, err, output.String())
	}
}

func runReadonlyBuildOutput(t *testing.T, ctx context.Context, engine *sandbox.Docker, dataDir, scope string) {
	t.Helper()
	workspace := volumeWorkspace(dataDir, scope)
	spec := sandbox.Spec{
		Workspace:         workspace,
		WorkspaceReadOnly: true,
		Artifacts:         filepath.Join(workspace, "builds"),
		Argv: []string{"sh", "-c",
			"if touch /workspace/forbidden 2>/dev/null; then exit 33; fi; " +
				"touch /output/artifact-" + scope + " && test -e /workspace/builds/artifact-" + scope +
				" && echo READONLY_OUTPUT_OK"},
	}
	var output bytes.Buffer
	result, err := engine.Run(ctx, spec, &output)
	if err != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "READONLY_OUTPUT_OK") {
		t.Fatalf("read-only artifact mount %s: code=%d err=%v output=%q", scope, result.ExitCode, err, output.String())
	}
}

func volumeWorkspace(dataDir, scope string) string {
	return filepath.Join(dataDir, "workspaces", scope)
}

func assertOutsideDataRejected(t *testing.T, ctx context.Context, engine *sandbox.Docker, outsideDir string) {
	t.Helper()
	_, err := engine.Run(ctx, sandbox.Spec{
		Workspace: outsideDir,
		Argv:      []string{"true"},
	}, io.Discard)
	if err == nil {
		t.Fatal("workspace outside the configured data directory was accepted")
	}
}
