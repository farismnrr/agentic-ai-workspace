package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestCleanPathStaysInsideRoot(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                  ".",
		".":                 ".",
		"/":                 ".",
		"src/main.go":       "src/main.go",
		"/src/../main.go":   "main.go",
		"../../etc/passwd":  "etc/passwd",
		`..\..\windows\ini`: "windows/ini",
	}

	for input, want := range cases {
		got, err := domain.CleanPath(input)
		if err != nil || got != want {
			t.Errorf("CleanPath(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestCleanPathDeniesSecrets(t *testing.T) {
	t.Parallel()

	for _, input := range []string{".env", "app/.env.local", ".git/config", "keys/server.pem", "x/id_rsa", "a\x00b"} {
		if _, err := domain.CleanPath(input); err == nil {
			t.Errorf("%q must be denied", input)
		}
	}
}
