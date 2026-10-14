package domain

// Scopes a client can be granted on the coding space.
const (
	ScopeRead  = "coding:read"
	ScopeWrite = "coding:write"
	ScopeExec  = "coding:exec"
)

// Group clusters tools that share a risk level; the app shows them together.
type Group string

// Tool groups.
const (
	GroupRead  Group = "read"
	GroupWrite Group = "write"
	GroupExec  Group = "exec"
)

// ToolInfo describes one MCP tool. It is the single catalogue the MCP server,
// the app's settings, the consent text and the permission defaults derive from.
type ToolInfo struct {
	Name        string
	Title       string
	Description string
	Scope       string
	Group       Group
	ReadOnly    bool
	DefaultOn   bool
}

func readTool(name, title, description string) ToolInfo {
	return ToolInfo{
		Name: name, Title: title, Description: description,
		Scope: ScopeRead, Group: GroupRead, ReadOnly: true, DefaultOn: true,
	}
}

func writeTool(name, title, description string) ToolInfo {
	return ToolInfo{Name: name, Title: title, Description: description, Scope: ScopeWrite, Group: GroupWrite}
}

// The tools the coding space offers.
var (
	ToolListFiles = readTool("list_files", "List files",
		"List the files and folders in a directory of the coding workspace.")
	ToolFindFiles = readTool("find_files", "Find files",
		"Find files by name with a glob pattern such as **/*.go or src/*.ts.")
	ToolReadFile = readTool("read_file", "Read a file",
		"Read a text file, or only lines start_line to end_line (large files are cut short).")
	ToolSearchCode = readTool("search_code", "Search code",
		"Find lines containing some text across the coding workspace.")
	ToolListTasks = readTool("list_tasks", "List tasks",
		"List the named tasks (such as tests) that run_task can run.")
	ToolWhoami = readTool("whoami", "Current account", "Return the signed-in account's stable id.")

	ToolEditFile = writeTool("edit_file", "Edit a file",
		"Replace old_text with new_text in a file. Fails if old_text is missing or not unique unless replace_all is set.")
	ToolWriteFile = writeTool("write_file", "Write a file",
		"Create a file, or replace its whole content. Missing folders are created.")
	ToolDeleteFile = writeTool("delete_file", "Delete a file", "Delete one file (never a folder).")
	ToolMoveFile   = writeTool("move_file", "Move a file", "Move or rename a file. It never overwrites an existing path.")

	ToolRunTask = ToolInfo{
		Name: "run_task", Title: "Run a task", Scope: ScopeExec, Group: GroupExec,
		Description: "Run one of the named tasks from list_tasks and return its exit code and output.",
	}
)

// Tools lists every tool in display order.
func Tools() []ToolInfo {
	return []ToolInfo{
		ToolListFiles, ToolFindFiles, ToolReadFile, ToolSearchCode, ToolListTasks, ToolWhoami,
		ToolEditFile, ToolWriteFile, ToolDeleteFile, ToolMoveFile, ToolRunTask,
	}
}
