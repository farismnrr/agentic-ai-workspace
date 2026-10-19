package domain_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

func TestAProjectIDFollowsTheRulesOfGoogleCloud(t *testing.T) {
	cases := map[string]bool{
		"toko-kopi": true, "abcdef": true, "a1-b2-c3": true, strings.Repeat("a", 30): true,
		"abcde": false, strings.Repeat("a", 31): false, "1toko-kopi": false, "toko-kopi-": false,
		"Toko-Kopi": false, "toko_kopi": false, "toko kopi": false, "-toko-kopi": false,
	}

	for id, want := range cases {
		if got := domain.ValidSlug(id); got != want {
			t.Errorf("%q: valid %v, want %v", id, got, want)
		}
	}
}

func TestTheSuggestedIDFromAnyNameIsValid(t *testing.T) {
	cases := map[string]string{
		"Toko Kopi":        "toko-kopi",
		"  My Cool App!! ": "my-cool-app",
		"2024 plans":       "p-2024-plans",
		"Hi":               "hi-app",
		"日本語":              "project",
		"":                 "project",
		"a/b\\c..d e f":    "a-b-c-d-e-f",
	}

	for name, want := range cases {
		got := domain.SlugFromName(name)
		if got != want || !domain.ValidSlug(got) {
			t.Errorf("%q: got %q, want %q", name, got, want)
		}
	}

	long := domain.SlugFromName(strings.Repeat("word ", 20))
	if !domain.ValidSlug(long) {
		t.Errorf("a long name must still make a valid ID: %q", long)
	}
}

func TestANumberedIDStaysWithinTheLimit(t *testing.T) {
	if got := domain.WithNumber("toko-kopi", 2); got != "toko-kopi-2" {
		t.Errorf("got %q", got)
	}

	full := strings.Repeat("a", 30)
	if got := domain.WithNumber(full, 12); len(got) != 30 || !domain.ValidSlug(got) || !strings.HasSuffix(got, "-12") {
		t.Errorf("got %q", got)
	}
}
