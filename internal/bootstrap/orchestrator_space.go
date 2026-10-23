package bootstrap

import spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"

const (
	orchestratorRead = "orchestrator:read"
	orchestratorRun  = "orchestrator:run"
)

func orchestratorSpace() spacedomain.Space {
	return spacedomain.Space{
		Slug: "orchestrator", Name: "Orchestrator", Entry: orchestratorRead,
		SupportsApprovals: false,
		Scopes: []spacedomain.Scope{
			{ID: orchestratorRead, Description: "Read the status of your Orchestrator tasks"},
			{ID: orchestratorRun, Description: "Start, continue and retry your Orchestrator tasks"},
		},
		Tools: []spacedomain.Tool{
			{
				Name: "start_task", Title: "Start a team task", Description: "Start a new task through your team",
				Scope: orchestratorRun, Group: "write",
			},
			{
				Name: "get_task", Title: "Get task status", Description: "Read a task's current status and open it in the app",
				Scope: orchestratorRead, Group: "read", ReadOnly: true, DefaultOn: true,
			},
			{
				Name: "send_message", Title: "Send a task message", Description: "Continue an eligible team task",
				Scope: orchestratorRun, Group: "write",
			},
			{
				Name: "retry_task", Title: "Retry a failed task", Description: "Retry an eligible failed team task",
				Scope: orchestratorRun, Group: "write",
			},
		},
	}
}
