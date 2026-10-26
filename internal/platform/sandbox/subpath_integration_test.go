package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Stage 7 uses the actual volume Subpath mount builder used by a deployed app.
// The test must run against a disposable volume prepopulated by stage7-docker.sh.
func TestStage7ActualVolumeSubpathsAreIsolated(t *testing.T) {
	volume, image := os.Getenv("STAGE7_VOLUME"), os.Getenv("STAGE7_IMAGE")
	if volume == "" || image == "" {
		t.Skip("STAGE7_VOLUME and STAGE7_IMAGE required for Docker integration")
	}

	cfg := Config{
		Image: image, DataVolume: volume, DataDir: "/data",
		UID: 65532, GID: 65532, Limits: DefaultLimits(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Docker must be available for Stage 7: %v", err)
	}

	for _, scope := range []string{"a", "b"} {
		other := "a"
		if scope == "a" {
			other = "b"
		}
		spec := Spec{
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
		result, runErr := engine.Run(ctx, spec, &output)
		if runErr != nil || result.ExitCode != 0 || !strings.Contains(output.String(), "SUBPATH_OK") {
			t.Fatalf("scope %s: code=%d err=%v output=%q", scope, result.ExitCode, runErr, output.String())
		}
	}

	if _, err := cfg.folderMount("/data/../outside", Workspace, false); !errors.Is(err, errOutsideData) {
		t.Fatalf("parent-directory escape accepted: %v", err)
	}
}
