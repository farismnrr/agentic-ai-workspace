package bootstrap

import (
	"context"
	"fmt"
	"strings"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func listFiles(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	listing, err := s.Reader.List(in.Path)
	if err != nil {
		return failure(err)
	}

	var out strings.Builder

	for _, entry := range listing.Entries {
		if entry.Dir {
			fmt.Fprintf(&out, "%s/\n", entry.Path)
		} else {
			fmt.Fprintf(&out, "%s (%d bytes)\n", entry.Path, entry.Size)
		}
	}

	if listing.Truncated {
		out.WriteString("(more entries exist than are shown)\n")
	}

	return taskapp.ToolResult{Output: orNothing(out.String(), "The folder is empty.")}
}

func findFiles(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	found, err := s.Reader.Find(in.Pattern, in.Path)
	if err != nil {
		return failure(err)
	}

	text := strings.Join(found.Paths, "\n")
	if found.Truncated {
		text += "\n(more files match than are shown)"
	}

	return taskapp.ToolResult{Output: orNothing(text, "No files match.")}
}

func readFile(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	file, err := s.Reader.ReadLines(in.Path, in.StartLine, in.EndLine)
	if err != nil {
		return failure(err)
	}

	text := file.Content
	if file.Truncated {
		text += "\n(the file is longer; the rest is cut off)"
	}

	return taskapp.ToolResult{Output: orNothing(text, "The file is empty.")}
}

func searchCode(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	result, err := s.Reader.Search(strings.TrimSpace(in.Query), in.Path, in.CaseSensitive)
	if err != nil {
		return failure(err)
	}

	var out strings.Builder

	for _, match := range result.Matches {
		fmt.Fprintf(&out, "%s:%d: %s\n", match.Path, match.Line, match.Text)
	}

	if result.Truncated {
		out.WriteString("(more matches exist than are shown)\n")
	}

	return taskapp.ToolResult{Output: orNothing(out.String(), "No matches.")}
}

func listTasks(_ context.Context, s codingmcp.Services, _ toolArgs) taskapp.ToolResult {
	tasks, err := s.Tasks.List()
	if err != nil {
		return failure(err)
	}

	var out strings.Builder
	for _, task := range tasks {
		fmt.Fprintf(&out, "%s: %s\n", task.Name, task.Description)
	}

	return taskapp.ToolResult{Output: orNothing(out.String(), "No tasks are set up.")}
}

func orNothing(text, empty string) string {
	if strings.TrimSpace(text) == "" {
		return empty
	}

	return text
}
