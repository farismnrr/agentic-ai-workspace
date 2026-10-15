package application

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTheTestedLineCountsChecksAndNeverListsCommands(t *testing.T) {
	t.Parallel()

	long := "go test ./... && GO111MODULE=off go test main.go -run 'a very long command line that nobody wants to read'"
	pass := domain.RunInfo{Name: long}
	fail := domain.RunInfo{Name: long, ExitCode: 1}
	slow := domain.RunInfo{Name: long, TimedOut: true}

	cases := []struct {
		want string
		runs []domain.RunInfo
	}{
		{"Nothing was run to check it", nil},
		{"Ran 1 check, and it passed", []domain.RunInfo{pass}},
		{"Ran 1 check, and it failed", []domain.RunInfo{fail}},
		{"Ran 1 check, and it failed", []domain.RunInfo{slow}},
		{"Ran 3 checks, all passed", []domain.RunInfo{pass, pass, pass}},
		{"Ran 2 checks, all failed", []domain.RunInfo{fail, slow}},
		{"Ran 3 checks: 2 passed, 1 failed", []domain.RunInfo{pass, fail, pass}},
	}

	for _, c := range cases {
		got := testedSummary(c.runs)
		if got != c.want || strings.Contains(got, "go test") {
			t.Fatalf("got %q, want %q", got, c.want)
		}
	}
}
