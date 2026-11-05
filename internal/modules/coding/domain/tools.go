package domain

// Scopes a client can be granted on the coding space.
const (
	ScopeRead    = "coding:read"
	ScopeWrite   = "coding:write"
	ScopeExec    = "coding:exec"
	ScopeApprove = "coding:approve"
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

// ToolMarimoExecute invokes the pinned marimo-pair skill in a one-shot sandbox.
var ToolMarimoExecute = ToolInfo{
	Name: "marimo_execute", Title: "Execute MoLab notebook code", Scope: ScopeExec, Group: GroupExec,
	Description: "Run Python in MoLab using the temporary URL and token from this chat. Requires active work_id with the Developer role and coding:exec permission. Reuse one task; no pairing form or stored credentials.",
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
	ToolWhoami       = readTool("whoami", "Current account", "Return the signed-in account's stable id.")
	ToolListProjects = readTool("list_projects", "List projects",
		"List the person's projects with their Project IDs, and which one is yours by default. Give a Project ID as "+
			"project to any other tool to work in that project, for example to read its files or start its web app; "+
			"leave project out to work in your own project. Ask the person which project they mean when it is not clear.")
	ToolLogMessage = ToolInfo{
		Name: "log_message", Title: "Save the conversation", Scope: ScopeRead, Group: GroupRead, DefaultOn: true,
		Description: "Save conversation messages in the person's task thread. Call with role \"user\" for the request. " +
			"For tracked work, call with role \"assistant\" for your summary, then call complete_work. Do not call " +
			"log_message after complete_work. A work_id is required; use begin_work with an enabled role " +
			"for tracked work. Never include passwords or keys.",
	}
	// External work lifecycle tools share log_message's existing read switch.
	ToolBeginWork = workTool("begin_work", "Begin work",
		"Start or resume ONE tracked work session for the current user issue. Reuse its work_id "+
			"through all roles; use set_role to change roles. A second begin_work automatically resumes the "+
			"open issue in the same app/project. Use new_task=true ONLY when the user explicitly starts "+
			"an independent issue. Supply an enabled non-lead role key.")
	ToolSetRole = workTool("set_role", "Set work role",
		"Switch the active role for this tracked task. Each role has different tool permissions; use the same work_id.")
	ToolUpdatePlan       = workTool("update_plan", "Update work plan", "Save the steps for this work session.")
	ToolSaveResearch     = workTool("save_research", "Save research", "Save research notes and their sources.")
	ToolSaveRequirements = workTool("save_requirements", "Save requirements", "Save the requirements document.")
	ToolReportProgress   = workTool("report_progress", "Report progress", "Record self-reported progress and claims. In owner-enabled Deep Review, Security must use this tool to submit a fresh Verdict: PASS report with Security Assessment, Architecture Assessment (if affected), Assessment naming changed paths, and exact server-recorded critical commands under Checks: before work can complete.")
	ToolCompleteWork     = workTool("complete_work", "Complete work",
		"Finish or block this work and summarize verified evidence. Call after saving your assistant summary with "+
			"log_message; do not call log_message after complete_work. In owner-enabled Deep Review, completion after code changes is blocked without a fresh server-attributed Security review and any required recorded checks. Use outcome blocked when verification cannot pass.")
	ToolGetWorkState = workTool("get_work_state", "Get work state", "Read the compact state of a work session.")
	ToolApprovePlan  = ToolInfo{
		Name: "approve_plan", Title: "Approve plans", Scope: ScopeApprove, Group: GroupExec,
		DefaultOn:   false,
		Description: "Lets a connected assistant approve plans and start the Orchestrator without asking the person.",
	}

	ToolEditFile = writeTool("edit_file", "Edit a file",
		"Replace old_text with new_text in a file. Fails if old_text is missing or not unique unless replace_all is set.")
	ToolWriteFile = writeTool("write_file", "Write a file",
		"Create a file, or replace its whole content. Missing folders are created.")
	ToolDeleteFile = writeTool("delete_file", "Delete a file", "Delete one file (never a folder).")
	ToolMoveFile   = writeTool("move_file", "Move a file", "Move or rename a file. It never overwrites an existing path.")

	ToolRunTask = ToolInfo{
		Name: "run_task", Title: "Run a task", Scope: ScopeExec, Group: GroupExec,
		Description: "Run an optional owner-configured named QA check (kind check or omitted), not a build.",
	}
	ToolVerify = ToolInfo{
		Name: "verify", Title: "Run QA verification", Scope: ScopeExec, Group: GroupExec,
		Description: "QA role: run test, lint and typecheck commands as argv directly in a read-only, credential-free sandbox. No tasks.json required.",
	}
	ToolRunBuild = ToolInfo{
		Name: "run_build", Title: "Run a build", Scope: ScopeExec, Group: GroupExec,
		Description: "Run an owner-configured kind=build task in a disposable writable copy of eligible source; /workspace stays read-only. Write final files to BUILD_OUT=/output; artifacts persist in .build-output-*/ and are not published to Sites.",
	}
	ToolRunArtifact = ToolInfo{
		Name: "run_artifact", Title: "Generate a document or media asset", Scope: ScopeExec, Group: GroupExec,
		Description: "Run only an owner-configured kind=artifact task for document/media output. Source is read-only; generated files persist in .artifact-output-*/.",
	}
	ToolGit = ToolInfo{
		Name: "git", Title: "Git", Scope: ScopeExec, Group: GroupExec,
		Description: "Server-mediated Git tool for Researcher, Developer and DevOps. Pass subcommand and arguments as an args array without the git prefix (e.g. [\"push\", \"origin\", \"dev\"]). Use this tool, NOT Bash or shell git, for GitHub operations in a restricted teammate session: only the server tool receives project-owner GitHub authorization and egress. A CONNECT_GITHUB_REQUIRED error means the owner must connect GitHub in Masih Awam Settings. Respect repository and non-force-push rules.",
	}
	ToolGh = ToolInfo{
		Name: "gh", Title: "GitHub CLI", Scope: ScopeExec, Group: GroupExec,
		Description: "Server-mediated GitHub CLI for Developer, QA or DevOps; send subcommand and arguments as an args array without the gh prefix. For restricted teammate sessions use THIS tool, not shell gh: the vendor CLI sandbox does not hold GitHub credentials or GitHub egress. Actions remain role-authorized by category.",
	}
)

func workTool(name, title, description string) ToolInfo {
	return ToolInfo{
		Name: name, Title: title, Description: description, Scope: ScopeRead, Group: GroupRead,
		ReadOnly: name == "get_work_state",
	}
}

// ToolRunCommand runs any command in the sandbox, so the agent can use the installed languages.
var ToolRunCommand = ToolInfo{
	Name: "run_command", Title: "Run a command", Scope: ScopeExec, Group: GroupExec,
	Description: "Run a shell command in the isolated sandbox (bash; Python, Node, Go, Rust and Java are installed; " +
		"only package registries can be reached) and return its exit code and output. Source stays read-only; " +
		"DevOps builds must use an enabled owner-configured run_build task. Do not run shell git push/fetch or gh here: " +
		"use the server-mediated git/gh tools for authorized GitHub access.",
}

// Tools lists every tool in display order.
func Tools() []ToolInfo {
	return []ToolInfo{
		ToolListFiles, ToolFindFiles, ToolReadFile, ToolSearchCode, ToolListTasks, ToolWhoami, ToolListProjects,
		ToolLogMessage,
		ToolEditFile, ToolWriteFile, ToolDeleteFile, ToolMoveFile, ToolRunTask, ToolVerify, ToolRunBuild, ToolRunArtifact,
		ToolGit, ToolGh, ToolRunCommand, ToolMarimoExecute, ToolScreenshotSession, ToolBrowserTest,
		ToolReadBrowserScreenshot,
		ToolApprovePlan,
	}
}
