package runner

import (
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
