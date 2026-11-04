package application

import (
	"strings"
	"testing"
)

func TestPreActionToolSOPRespectsEnabledRoleInventory(t *testing.T) {
	tests := []struct {
		name string
		role string
		enabled []string
		want []string
		absent []string
	}{
		{
			name: "devops has server git and CI", role: "devops",
			enabled: []string{"gh", "git", "ci_logs"},
			want: []string{"ENABLED project tools: ci_logs, gh, git", "Git / GitHub", "CI: retrieve failed job logs", "NEVER use shell git/gh"},
			absent: []string{"ENABLED project tools: browser_test"},
		},
		{
			name: "developer with disabled git", role: "developer",
			enabled: []string{"read_file", "write_file"},
			want: []string{"ENABLED project tools: read_file, write_file", "git/gh server tools are NOT enabled", "report the precise blocker"},
			absent: []string{"ENABLED project tools: git"},
		},
		{
			name: "QA Playwright", role: "qa",
			enabled: []string{"read_browser_screenshot", "browser_test", "verify"},
			want: []string{"ENABLED project tools: browser_test, read_browser_screenshot, verify", "Browser / Playwright E2E", "use browser_test", "read_browser_screenshot", "running project Site"},
			absent: []string{"ENABLED project tools: git"},
		},
		{
			name: "no enabled tools", role: "developer",
			want: []string{"No server project tools are enabled", "report the missing permission"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preActionToolSOP(tt.role, tt.enabled)
			for _, token := range tt.want {
				if !strings.Contains(got, token) {
					t.Errorf("SOP missing %q", token)
				}
			}
			for _, token := range tt.absent {
				if strings.Contains(got, token) {
					t.Errorf("SOP unexpectedly advertises %q", token)
				}
			}
			if !strings.Contains(got, "instead of trying random transports") {
				t.Fatal("SOP must explicitly forbid fallback trial-and-error")
			}
		})
	}
}

func TestLeadToolSOPNamesCorrectOwnersBeforeAction(t *testing.T) {
	for _, required := range []string{
		"Before delegating", "Developer or DevOps", "Git", "QA", "browser_test",
		"read_browser_screenshot", "start_site", "ci_logs", "missing/disabled",
	} {
		if !strings.Contains(preActionLeadSOP, required) {
			t.Fatalf("lead SOP missing %q", required)
		}
	}
}
