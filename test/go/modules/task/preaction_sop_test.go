package task_test

import (
	"strings"
	"testing"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestPreActionToolSOPRespectsEnabledRoleInventory(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		enabled []string
		want    []string
		absent  []string
	}{
		{
			name: "devops has server git and CI", role: "devops",
			enabled: []string{"gh", "git", "ci_logs"},
			want:    []string{"ENABLED project tools: ci_logs, gh, git", "Git / GitHub", "CI: retrieve failed job logs", "NEVER use shell git/gh"},
			absent:  []string{"ENABLED project tools: browser_test"},
		},
		{
			name: "developer with disabled git", role: "developer",
			enabled: []string{"read_file", "write_file"},
			want:    []string{"ENABLED project tools: read_file, write_file", "git/gh server tools are NOT enabled", "report the precise blocker"},
			absent:  []string{"ENABLED project tools: git"},
		},
		{
			name: "QA Playwright", role: "qa",
			enabled: []string{"read_browser_screenshot", "browser_test", "verify"},
			want:    []string{"ENABLED project tools: browser_test, read_browser_screenshot, verify", "Browser / Playwright E2E", "before calling browser_test", "actions [{\"action\":\"goto\",\"path\":\"\"}]", "NOT '/'", "read_browser_screenshot", "DevOps has started"},
			absent:  []string{"ENABLED project tools: git"},
		},
		{
			name: "no enabled tools", role: "developer",
			want: []string{"No server project tools are enabled", "report the missing permission"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := taskapp.PreActionToolSOP(tt.role, tt.enabled)
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
		if !strings.Contains(taskapp.PreActionLeadSOP, required) {
			t.Fatalf("lead SOP missing %q", required)
		}
	}
}

func TestPreActionToolSOPShowsActualBridgedProjectFileTools(t *testing.T) {
	for _, tc := range []struct {
		name    string
		role    string
		enabled []string
		want    []string
		absent  []string
	}{
		{
			name:    "researcher can read source",
			role:    "researcher",
			enabled: []string{"list_files", "find_files", "read_file", "search_code", "get_work_state", "report_progress"},
			want:    []string{"ACTUAL TEAM MCP TOOLS: find_files, list_files, read_file, search_code", "Project source inspection: prefer team MCP read_file", "relative/file.ext"},
			absent:  []string{"ACTUAL TEAM MCP TOOLS: get_work_state", "ACTUAL TEAM MCP TOOLS: report_progress", "team read_file is NOT available"},
		},
		{
			name:    "disabled read cannot be claimed",
			role:    "researcher",
			enabled: []string{"list_files", "get_work_state"},
			want:    []string{"ACTUAL TEAM MCP TOOLS: list_files", "team read_file is NOT available", "Settings > Spaces & tools"},
			absent:  []string{"Project source inspection: prefer team MCP read_file", "ACTUAL TEAM MCP TOOLS: get_work_state"},
		},
		{
			name:    "developer has authorized writes",
			role:    "developer",
			enabled: []string{"edit_file", "read_file", "write_file"},
			want:    []string{"ACTUAL TEAM MCP TOOLS: edit_file, read_file, write_file"},
			absent:  []string{"Editing is unavailable"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := taskapp.PreActionToolSOP(tc.role, tc.enabled)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q", want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(got, absent) {
					t.Errorf("unexpected claim %q", absent)
				}
			}
		})
	}
}
