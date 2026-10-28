package sandbox_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
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

	cfg := sandbox.Config{
		Image: image, DataVolume: volume, DataDir: "/data",
		UID: 65532, GID: 65532, Limits: sandbox.DefaultLimits(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine, err := sandbox.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Docker must be available for volume isolation: %v", err)
	}

	runOwnerScope(t, ctx, engine, "a", "b")
	runOwnerScope(t, ctx, engine, "b", "a")
	assertOutsideDataRejected(t, ctx, engine)
}

func runOwnerScope(t *testing.T, ctx context.Context, engine *sandbox.Docker, scope, other string) {
	t.Helper()
	spec := sandbox.Spec{
		Workspace: "/data/workspaces/" + scope,
		Cache:     "/data/caches/" + scope,
		Argv: []string{
			"sh", "-c",
			"test -e /workspace/marker-" + scope +
				" && test ! -e /workspace/marker-" + other +
				" && test -e /cache/cache-" + scope +
				" && test ! -e /cache/cache-" + other +
				" && touch /workspace/wrote-" + scope +
				" && echo SUBPATH_OK",
		},
	}
	var output bytes.Buffer
	result, err := engine.Run(ctx, spec, &output)
	if err != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "SUBPATH_OK") {
		t.Fatalf("scope %s: code=%d err=%v output=%q", scope, result.ExitCode, err, output.String())
	}
}

func assertOutsideDataRejected(t *testing.T, ctx context.Context, engine *sandbox.Docker) {
	t.Helper()
	_, err := engine.Run(ctx, sandbox.Spec{
		Workspace: "/data/../outside",
		Argv:      []string{"true"},
	}, io.Discard)
	if !errors.Is(err, sandbox.ErrOutsideData) {
		t.Fatalf("parent-directory escape accepted: %v", err)
	}
}
