package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// OrchestratorProjects lists the caller's projects for the external task picker.
type OrchestratorProjects interface {
	List(ctx context.Context, userID string) ([]ProjectSummary, error)
}

// ProjectSummary is the small project shape exposed by Orchestrator MCP.
type ProjectSummary struct {
	ID   string
	Name string
}

// OrchestratorTasks lists eligible team tasks for the caller and selected project.
type OrchestratorTasks interface {
	ListOrchestrator(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error)
}
