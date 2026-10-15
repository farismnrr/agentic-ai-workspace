package runner

import (
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func TestGitEnvironmentSignsInWithoutWritingTheToken(t *testing.T) {
	env := gitEnvironment(Git{Token: "ghu_secret", Name: "Ada", Email: "1+ada@users.noreply.github.com"})

	for _, want := range []string{
		"GH_TOKEN=ghu_secret", "GIT_AUTHOR_NAME=Ada", "GIT_COMMITTER_EMAIL=1+ada@users.noreply.github.com",
		"GIT_CONFIG_KEY_0=credential.https://github.com.helper",
	} {
		if !slices.Contains(env, want) {
			t.Errorf("missing %q in %v", want, env)
		}
	}

	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_CONFIG_VALUE_0=") && strings.Contains(entry, "ghu_secret") {
			t.Fatal("the credential helper must read the token from the environment, not embed it")
		}
	}
}

func TestPlainEnvironmentHasNoToken(t *testing.T) {
	for _, entry := range environment(sandbox.Spec{Egress: "/run/egress"}) {
		if strings.Contains(entry, "GH_TOKEN") || strings.Contains(entry, "ghu_") {
			t.Fatalf("web apps and unconnected people must never see a token: %q", entry)
		}
	}
}
