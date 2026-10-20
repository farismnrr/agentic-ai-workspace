package bootstrap

import (
	"slices"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// claudeToolFor says which of Claude Code's own tools do what each of the app's project tools does.
func claudeToolFor() map[string][]string {
	return map[string][]string{
		codingdomain.ToolListFiles.Name:  {"LS", "Glob"},
		codingdomain.ToolFindFiles.Name:  {"Glob"},
		codingdomain.ToolReadFile.Name:   {"Read"},
		codingdomain.ToolSearchCode.Name: {"Grep"},
		codingdomain.ToolEditFile.Name:   {"Edit", "MultiEdit"},
		codingdomain.ToolWriteFile.Name:  {"Write"},
		codingdomain.ToolRunCommand.Name: {"Bash"},
		codingdomain.ToolRunTask.Name:    {"Bash"},
		codingdomain.ToolWebSearch.Name:  {"WebSearch"},
		codingdomain.ToolFetchPage.Name:  {"WebFetch"},
	}
}

// claudeTools are the tools of Claude Code a role may use: what its project tools map to, and nothing else.
func claudeTools(allowed []string) []string {
	mapping := claudeToolFor()
	names := []string{}

	for _, tool := range allowed {
		for _, own := range mapping[tool] {
			if !slices.Contains(names, own) {
				names = append(names, own)
			}
		}
	}

	slices.Sort(names)

	return names
}

// mayChange says whether the tools let a teammate change the project or run anything in it.
func mayChange(allowed []string) bool {
	for _, tool := range []string{
		codingdomain.ToolEditFile.Name, codingdomain.ToolWriteFile.Name, codingdomain.ToolDeleteFile.Name,
		codingdomain.ToolMoveFile.Name, codingdomain.ToolRunCommand.Name, codingdomain.ToolRunTask.Name,
	} {
		if slices.Contains(allowed, tool) {
			return true
		}
	}

	return false
}
