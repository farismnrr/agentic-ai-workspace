package domain_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

func TestLogRefNeedsARealRepoAndExactlyOneOfRunOrJob(t *testing.T) {
	cases := map[string]struct {
		ref  domain.LogRef
		want bool
	}{
		"run":          {domain.LogRef{Repo: "farismnrr/sensio-iot-new", RunID: 7}, true},
		"job":          {domain.LogRef{Repo: "a/b.c_d-e", JobID: 9}, true},
		"neither":      {domain.LogRef{Repo: "a/b"}, false},
		"both":         {domain.LogRef{Repo: "a/b", RunID: 1, JobID: 2}, false},
		"no slash":     {domain.LogRef{Repo: "ab", RunID: 1}, false},
		"path escape":  {domain.LogRef{Repo: "a/../../user", RunID: 1}, false},
		"query inject": {domain.LogRef{Repo: "a/b?x=1", RunID: 1}, false},
		"three parts":  {domain.LogRef{Repo: "a/b/c", RunID: 1}, false},
	}

	for name, c := range cases {
		if got := c.ref.Valid(); got != c.want {
			t.Errorf("%s: Valid() = %v, want %v", name, got, c.want)
		}
	}
}

func TestTailKeepsTheEndOfALogWithinBothLimits(t *testing.T) {
	log := "a\nb\nc\nd\ne\n"

	if got := domain.Tail(log, 2, 100); got != "d\ne" {
		t.Errorf("last two lines = %q", got)
	}

	if got := domain.Tail(strings.Repeat("x", 50)+"\nEND", 10, 6); got != "xx\nEND" {
		t.Errorf("byte limit must keep the end, got %q", got)
	}

	if got := domain.Tail("only", 5, 100); got != "only" {
		t.Errorf("short log = %q", got)
	}
}
