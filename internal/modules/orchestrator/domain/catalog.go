// Package domain defines the independent external Orchestrator MCP catalog.
package domain

import spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"

const (
	ScopeRead     = "orchestrator:read"
	ScopeTasks    = "orchestrator:tasks"
	ScopeDelegate = "orchestrator:delegate"
	ScopeFiles    = "orchestrator:files"
	ScopeApprove  = "orchestrator:approve"
)

func Space() spacedomain.Space {
	tool := func(name, title, scope, group string, readOnly bool) spacedomain.Tool {
		return spacedomain.Tool{
			Name: name, Title: title, Description: title, Scope: scope,
			Group: group, ReadOnly: readOnly, DefaultOn: readOnly,
		}
	}
	return spacedomain.Space{
		Slug: "orchestrator", Name: "Orchestrator", Entry: ScopeRead,
		Scopes: []spacedomain.Scope{
			{ID: ScopeRead, Description: "Read your existing projects, agents and tasks"},
			{ID: ScopeTasks, Description: "Create and manage external-led tasks"},
			{ID: ScopeDelegate, Description: "Delegate directly to your internal agents"},
			{ID: ScopeFiles, Description: "Save files to an existing project"},
			{ID: ScopeApprove, Description: "Approve a submitted external Orchestrator PRD on your behalf"},
		},
		Tools: []spacedomain.Tool{
			tool("list_projects", "List your existing projects", ScopeRead, "read", true),
			tool("list_agents", "List enabled internal worker agents", ScopeRead, "read", true),
			tool("list_tasks", "List dashboard tasks", ScopeRead, "read", true),
			tool("get_task", "Read a task and its current state", ScopeRead, "read", true),
			tool("get_task_activity", "Read a task's activity", ScopeRead, "read", true),
			tool("get_delegation", "Read one internal agent's job", ScopeRead, "read", true),
			tool("list_delegations", "List internal jobs on a primary task", ScopeRead, "read", true),
			tool("get_approval_state", "Read owner approval state", ScopeRead, "read", true),
			tool("list_files", "List project files", ScopeRead, "read", true),
			tool("create_task", "Create one external-led task", ScopeTasks, "write", false),
			tool("delegate_agent", "Run a configured internal worker directly", ScopeDelegate, "exec", false),
			tool("cancel_delegation", "Cancel a worker job", ScopeDelegate, "exec", false),
			tool("stop_task", "Stop an external-led task", ScopeTasks, "write", false),
			tool("complete_task", "Complete an external-led task", ScopeTasks, "write", false),
			tool("update_task_plan", "Update the external lead plan", ScopeTasks, "write", false),
			tool("save_task_research", "Save external lead research", ScopeTasks, "write", false),
			tool("save_task_requirements", "Save the external lead PRD", ScopeTasks, "write", false),
			tool("submit_plan", "Ask the owner to approve the PRD", ScopeTasks, "write", false),
			tool("approve_plan", "Approve a submitted external Orchestrator PRD (requires separate owner opt-in)", ScopeApprove, "exec", false),
			tool("save_file", "Save a text file in an existing project", ScopeFiles, "write", false),
			tool("start_upload", "Start a resumable binary file transfer", ScopeFiles, "write", false),
			tool("upload_chunk", "Append a bounded binary file chunk", ScopeFiles, "write", false),
			tool("upload_status", "Read upload transfer status", ScopeFiles, "write", false),
			tool("finish_upload", "Verify and publish a transferred file", ScopeFiles, "write", false),
			tool("cancel_upload", "Cancel an unfinished transfer", ScopeFiles, "write", false),
		},
	}
}
