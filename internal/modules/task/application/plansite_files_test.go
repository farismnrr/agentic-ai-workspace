package application

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestBridgedToolsExposesEnabledProjectFilesOnly(t *testing.T) {
	defs := []domain.ToolDef{
		{Name: "list_files", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "find_files", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "read_file", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "search_code", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "edit_file", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "write_file", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "delete_file", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "git", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "browser_test", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "run_command", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "get_work_state", Schema: json.RawMessage(`{"type":"object"}`)},
	}
	tests := []struct {
		name string
		allowed []string
		want []string
	}{
		{
			name: "researcher read access",
			allowed: []string{"list_files", "find_files", "read_file", "search_code", "git", "get_work_state"},
			want: []string{"list_files", "find_files", "read_file", "search_code", "git"},
		},
		{
			name: "developer write access",
			allowed: []string{"list_files", "read_file", "edit_file", "write_file", "get_work_state"},
			want: []string{"list_files", "read_file", "edit_file", "write_file"},
		},
		{
			name: "disabled file read stays disabled",
			allowed: []string{"list_files", "find_files", "get_work_state"},
			want: []string{"list_files", "find_files"},
		},
		{
			name: "QA role can read and test browser",
			allowed: []string{"read_file", "search_code", "browser_test"},
			want: []string{"read_file", "search_code", "browser_test"},
		},
		{
			name: "empty owner switches",
			allowed: nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bridgedTools(defs, tt.allowed)
			names := make([]string, 0, len(got))
			for _, tool := range got {
				names = append(names, tool.Name)
			}
			if !slices.Equal(names, tt.want) {
				t.Fatalf("bridged team tool names = %v; want %v", names, tt.want)
			}
			for _, tool := range got {
				if !json.Valid(tool.Schema) {
					t.Fatalf("bridge schema invalid for %s", tool.Name)
				}
			}
		})
	}
}

func TestTeamBridgeRejectsHiddenToolBeforeExecution(t *testing.T) {
	session := &siteSession{
		tools: []CLITool{{Name: "list_files"}},
	}
	for _, hidden := range []string{"read_file", "write_file", "git", "run_command", "get_work_state"} {
		got := session.call(context.Background(), hidden, json.RawMessage(`{}`))
		if !got.Failed || !strings.Contains(got.Text, "no enabled team tool") {
			t.Fatalf("hidden %s was not rejected at bridge boundary: %+v", hidden, got)
		}
	}
}

func TestReadOnlyTeamFileToolsAreCatalogued(t *testing.T) {
	for _, name := range []string{"list_files", "find_files", "read_file", "search_code", "list_tasks", "view_image"} {
		if !slices.Contains(siteToolNames, name) {
			t.Fatalf("required file tool not bridged: %s", name)
		}
	}
}
