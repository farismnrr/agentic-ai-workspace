package runner

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestReadOnlyCheckRequiresServerEnforcedNonWritableCommand(t *testing.T) {
	readonly := domain.WithReadOnlyWorkspace(context.Background())
	both := domain.WithGitOnly(readonly)
	cases := []struct {
		name string
		ctx  context.Context
		kind string
		want bool
	}{
		{"normal command with server read-only context", readonly, "", true},
		{"named check with server read-only context", readonly, "check", true},
		{"ordinary task without read-only context", context.Background(), "", false},
		{"read-only context with GitOnly", both, "", false},
		{"build with writable artifact output", readonly, "build", false},
		{"artifact with writable output", readonly, "artifact", false},
		{"site task", readonly, "site", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := readOnlyCheck(tc.ctx, domain.Task{Kind: tc.kind})
			if got != tc.want {
				t.Fatalf("readOnlyCheck(kind=%q) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}
