package bootstrap

import (
	"fmt"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

const pathProp = `"path":{"type":"string","description":"Path relative to the project folder"}`

var toolSchemas = map[string]string{
	"list_files": `{"type":"object","properties":{` + pathProp + `}}`,
	"find_files": `{"type":"object","properties":{"pattern":{"type":"string","description":"Glob such as **/*.html"},` +
		pathProp + `},"required":["pattern"]}`,
	"read_file": `{"type":"object","properties":{` + pathProp + `,"start_line":{"type":"integer"},` +
		`"end_line":{"type":"integer"}},"required":["path"]}`,
	"search_code": `{"type":"object","properties":{"query":{"type":"string"},` + pathProp +
		`,"case_sensitive":{"type":"boolean"}},"required":["query"]}`,
	"list_tasks": `{"type":"object","properties":{}}`,
	"edit_file": `{"type":"object","properties":{` + pathProp + `,"old_text":{"type":"string"},` +
		`"new_text":{"type":"string"},"replace_all":{"type":"boolean"}},"required":["path","old_text","new_text"]}`,
	"write_file": `{"type":"object","properties":{` + pathProp + `,"content":{"type":"string"}},` +
		`"required":["path","content"]}`,
	"delete_file": `{"type":"object","properties":{` + pathProp + `},"required":["path"]}`,
	"move_file": `{"type":"object","properties":{"from":{"type":"string"},"to":{"type":"string"}},` +
		`"required":["from","to"]}`,
	"run_task": `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
	"run_command": `{"type":"object","properties":{"command":{"type":"string","description":"Shell command line"},` +
		`"timeout_seconds":{"type":"integer"}},"required":["command"]}`,
}

type describer func(in toolArgs, target string) taskapp.Described

// describers word each tool call: what was done, the same as a question, and the technical line.
var describers = map[string]describer{
	"list_files": func(in toolArgs, target string) taskapp.Described {
		return said("Looked at the files in "+target, "", "list_files "+in.Path)
	},
	"find_files": func(in toolArgs, _ string) taskapp.Described {
		return said(fmt.Sprintf("Looked for files matching %q", in.Pattern), "", "find_files "+in.Pattern)
	},
	"read_file": func(in toolArgs, target string) taskapp.Described {
		return said("Read "+target, "", "read_file "+in.Path)
	},
	"search_code": func(in toolArgs, _ string) taskapp.Described {
		return said(fmt.Sprintf("Searched for %q", in.Query), "", "search_code "+in.Query)
	},
	"list_tasks": func(toolArgs, string) taskapp.Described {
		return said("Checked which checks are set up", "", "list_tasks")
	},
	"edit_file": func(in toolArgs, target string) taskapp.Described {
		return said("Edited "+target, "May I edit "+target+"?", "edit_file "+in.Path)
	},
	"write_file": func(in toolArgs, target string) taskapp.Described {
		return said("Wrote "+target, "May I write "+target+"?", "write_file "+in.Path)
	},
	"delete_file": func(in toolArgs, target string) taskapp.Described {
		return said("Deleted "+target, "May I delete "+target+"? A copy is kept so it can be undone.", "delete_file "+in.Path)
	},
	"move_file": func(in toolArgs, _ string) taskapp.Described {
		return said(fmt.Sprintf("Moved %s to %s", in.From, in.To), fmt.Sprintf("May I move %s to %s?", in.From, in.To),
			"move_file "+in.From+" "+in.To)
	},
	"run_command": describeCommand,
	"run_task": func(in toolArgs, _ string) taskapp.Described {
		return said(fmt.Sprintf("Ran the %q check", in.Name), fmt.Sprintf("May I run the %q check?", in.Name),
			"run_task "+in.Name)
	},
}

func describeCommand(in toolArgs, _ string) taskapp.Described {
	return said("Ran a command: "+in.Command, "May I run this command? "+in.Command, "run_command "+in.Command)
}

func describe(tool string, in toolArgs) taskapp.Described {
	target := in.Path
	if target == "" {
		target = "the project"
	}

	if describer, ok := describers[tool]; ok {
		return describer(in, target)
	}

	if describer, ok := siteDescribers[tool]; ok {
		return describer(in, target)
	}

	if tool == toolDeployExtension {
		return describeExtension(in)
	}

	if tool == toolScreenshot {
		return describeShot(in)
	}
	if tool == toolScreenshotSession {
		return describeSessionAction(in)
	}

	if tool == toolViewImage {
		return describeView(in)
	}

	if codingdomain.IsWebTool(tool) {
		return describeWeb(tool, in)
	}

	return said("Used "+tool, "May I use "+tool+"?", tool)
}

func said(text, ask, technical string) taskapp.Described {
	if ask == "" {
		ask = text + "?"
	}

	return taskapp.Described{Text: text, Ask: ask, Technical: technical}
}
