package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type listInput struct {
	Target
	Path string `json:"path,omitempty" jsonschema:"directory relative to the workspace root; defaults to the root"`
}

type fileEntry struct {
	Path string `json:"path"`
	Type string `json:"type" jsonschema:"file or directory"`
	Size int64  `json:"size,omitempty" jsonschema:"size in bytes for files"`
}

type listOutput struct {
	Entries   []fileEntry `json:"entries"`
	Truncated bool        `json:"truncated" jsonschema:"true when more entries exist than were returned"`
}

type findInput struct {
	Target
	Pattern string `json:"pattern" jsonschema:"glob such as **/*.go or src/*.ts; ** spans folders"`
	Path    string `json:"path,omitempty" jsonschema:"directory to search in; defaults to the workspace root"`
}

type findOutput struct {
	Paths     []string `json:"paths"`
	Truncated bool     `json:"truncated" jsonschema:"true when there were more files than were returned"`
}

type readInput struct {
	Target
	Path      string `json:"path" jsonschema:"file path relative to the workspace root"`
	StartLine int    `json:"start_line,omitempty" jsonschema:"first line to return (1-based); defaults to the first line"`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"last line to return, inclusive; defaults to the last line"`
}

type readOutput struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	TotalLines int    `json:"total_lines"`
	Truncated  bool   `json:"truncated" jsonschema:"true when the file was longer than the limit and was cut short"`
}

type searchInput struct {
	Target
	Query         string `json:"query" jsonschema:"text to look for (matched literally, not as a regular expression)"`
	Path          string `json:"path,omitempty" jsonschema:"directory to search in; defaults to the workspace root"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" jsonschema:"match upper and lower case exactly"`
}

type searchHit struct {
	Path string `json:"path"`
	Text string `json:"text"`
	Line int    `json:"line"`
}

type searchOutput struct {
	Matches   []searchHit `json:"matches"`
	Truncated bool        `json:"truncated" jsonschema:"true when there were more matches than were returned"`
}

func (t *tools) addReadTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolListFiles), guarded(t.guard, domain.ToolListFiles,
		func(in listInput) string { return "list " + orRoot(in.Path) }, t.listFiles))
	sdk.AddTool(server, toolOf(domain.ToolFindFiles), guarded(t.guard, domain.ToolFindFiles,
		func(in findInput) string { return "find " + in.Pattern }, t.findFiles))
	sdk.AddTool(server, toolOf(domain.ToolReadFile), guarded(t.guard, domain.ToolReadFile, describeRead, t.readFile))
	sdk.AddTool(server, toolOf(domain.ToolSearchCode), guarded(t.guard, domain.ToolSearchCode,
		func(in searchInput) string { return "search " + strconv.Quote(in.Query) }, t.searchCode))
	sdk.AddTool(server, toolOf(domain.ToolListTasks), guarded(t.guard, domain.ToolListTasks,
		func(Target) string { return "list tasks" }, t.listTasks))
	sdk.AddTool(server, toolOf(domain.ToolWhoami), guarded(t.guard, domain.ToolWhoami,
		func(struct{}) string { return "whoami" }, t.whoami))
}

func orRoot(path string) string {
	if strings.TrimSpace(path) == "" {
		return "."
	}

	return path
}

func describeRead(in readInput) string {
	if in.StartLine == 0 && in.EndLine == 0 {
		return "read " + in.Path
	}

	return "read " + in.Path + ":" + strconv.Itoa(in.StartLine) + "-" + strconv.Itoa(in.EndLine)
}

func (t *tools) listFiles(ctx context.Context, in listInput) (listOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return listOutput{}, report{}, err
	}

	listing, err := svc.Reader.List(in.Path)
	if err != nil {
		return listOutput{}, report{}, fmt.Errorf("list: %w", err)
	}

	out := listOutput{Entries: make([]fileEntry, 0, len(listing.Entries)), Truncated: listing.Truncated}

	for _, entry := range listing.Entries {
		kind := "file"
		if entry.Dir {
			kind = "directory"
		}

		out.Entries = append(out.Entries, fileEntry{Path: entry.Path, Type: kind, Size: entry.Size})
	}

	return out, report{}, nil
}

func (t *tools) findFiles(ctx context.Context, in findInput) (findOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return findOutput{}, report{}, err
	}

	found, err := svc.Reader.Find(in.Pattern, in.Path)
	if err != nil {
		return findOutput{}, report{}, fmt.Errorf("find: %w", err)
	}

	return findOutput{Paths: append([]string{}, found.Paths...), Truncated: found.Truncated}, report{}, nil
}

func (t *tools) readFile(ctx context.Context, in readInput) (readOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return readOutput{}, report{}, err
	}

	file, err := svc.Reader.ReadLines(in.Path, in.StartLine, in.EndLine)
	if err != nil {
		return readOutput{}, report{}, fmt.Errorf("read: %w", err)
	}

	return readOutput{
		Path: file.Path, Content: file.Content, TotalLines: file.TotalLines, Truncated: file.Truncated,
	}, report{}, nil
}

func (t *tools) searchCode(ctx context.Context, in searchInput) (searchOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return searchOutput{}, report{}, err
	}

	result, err := svc.Reader.Search(strings.TrimSpace(in.Query), in.Path, in.CaseSensitive)
	if err != nil {
		return searchOutput{}, report{}, fmt.Errorf("search: %w", err)
	}

	out := searchOutput{Matches: make([]searchHit, 0, len(result.Matches)), Truncated: result.Truncated}
	for _, match := range result.Matches {
		out.Matches = append(out.Matches, searchHit{Path: match.Path, Line: match.Line, Text: match.Text})
	}

	return out, report{}, nil
}
