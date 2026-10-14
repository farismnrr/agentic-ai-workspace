package runner

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func TestEnvironmentPointsCachesAndProxyOnlyWhenMounted(t *testing.T) {
	bare := strings.Join(environment(sandbox.Spec{}), "\n")
	if strings.Contains(bare, "PROXY") || strings.Contains(bare, "/cache") {
		t.Fatalf("without extras the environment must not mention them:\n%s", bare)
	}

	full := environment(sandbox.Spec{Cache: "/x", Egress: "/y"})
	wanted := []string{"GOMODCACHE=/cache/go/mod", "CARGO_HOME=/cache/cargo", "HTTPS_PROXY=http://127.0.0.1:3128"}

	for _, want := range wanted {
		if !slices.Contains(full, want) {
			t.Errorf("missing %s in %v", want, full)
		}
	}
}

func TestWipeRemovesReadOnlyFolders(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	deep := filepath.Join(root, "mod", "pkg")

	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(deep, "a.go"), []byte("x"), 0o400); err != nil {
		t.Fatal(err)
	}

	// Go's module cache looks like this: folders nobody may write to.
	for _, dir := range []string{deep, filepath.Join(root, "mod")} {
		if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // mimics the read-only Go module cache
			t.Fatal(err)
		}
	}

	wipe(root)

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("the cache should be gone, got %v", err)
	}
}
