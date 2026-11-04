package application

import (
	"strings"
	"testing"
)

func TestBridgeVCSInstructionsAreScopedToServerAuthorizedTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    []string
		want     []string
		wantNone []string
	}{
		{name: "no vcs tools", tools: []string{"list_files"}, wantNone: []string{"GitHub access", "git with args", "gh with an args"}},
		{
			name: "researcher git only", tools: []string{"git", "read_file"},
			want: []string{"ONLY through the team's server tools", "server tool named git", "CONNECT_GITHUB_REQUIRED", "HTTP proxy 403", "non-force-push"},
			wantNone: []string{"server tool named gh"},
		},
		{
			name: "qa gh only", tools: []string{"gh", "verify"},
			want: []string{"server tool named gh", "no GitHub token", "not with curl"},
			wantNone: []string{"server tool named git"},
		},
		{
			name: "devops and developer", tools: []string{"git", "gh", "run_command"},
			want: []string{"server tool named git", "server tool named gh", "never", "HTTPS_PROXY"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bridgeVCSInstructions(tc.tools)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("bridge instructions missing %q", want)
				}
			}
			for _, unwant := range tc.wantNone {
				if strings.Contains(got, unwant) {
					t.Errorf("bridge instructions unexpectedly contain %q", unwant)
				}
			}
			if len(tc.want) == 0 && got != "" {
				t.Errorf("no VCS tools should produce no VCS instructions, got %q", got)
			}
		})
	}
}
