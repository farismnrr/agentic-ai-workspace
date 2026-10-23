package mcp

import (
	"context"
	"net/http"
	"strings"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service is the task application contract required by the Orchestrator tools.
type Service interface {
	ListOrchestratorProjects(ctx context.Context, userID string) ([]taskapp.ProjectSummary, error)
	ListOrchestratorTasks(ctx context.Context, userID, projectID string) ([]taskdomain.Task, error)
	StartOrchestrator(ctx context.Context, userID, projectID, prompt string) (taskdomain.Task, error)
	GetOrchestrator(ctx context.Context, userID, projectID, taskID string) (taskdomain.Task, error)
	SendOrchestratorMessage(ctx context.Context, userID, projectID, taskID, text string) error
	RetryOrchestrator(ctx context.Context, userID, projectID, taskID string) error
}

// Gate checks the user's switch for an Orchestrator tool.
type Gate interface {
	Allowed(ctx context.Context, userID, tool string) (bool, error)
}

// Activity records successful and refused tool calls.
type Activity interface {
	Record(ctx context.Context, entry Entry) error
}

// Entry describes one call for the shared activity log.
type Entry struct {
	UserID, Client, Project, Task, Tool, Summary, Detail string
	Outcome                                              string
}

// Options connect the application operations and access controls.
type Options struct {
	Gate                Gate
	Activity            Activity
	ResourceMetadataURL string
	TaskURLBase         string
}

// NewServer exposes only the four public Orchestrator operations.
func NewServer(service Service, opts Options) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "masih-awam-orchestrator", Version: "1.0.0"},
		&sdk.ServerOptions{Instructions: "Use these tools to start and follow team tasks. Open the returned task URL " +
			"in the app to answer questions, approve work, or accept results."})
	t := tools{
		service: service, gate: opts.Gate, activity: opts.Activity,
		metadataURL: opts.ResourceMetadataURL, taskURLBase: strings.TrimRight(opts.TaskURLBase, "/"),
	}
	t.addTools(server)

	return server
}

// HTTPHandler serves stateless JSON MCP requests, following the Coding Space transport convention.
func HTTPHandler(service Service, opts Options) http.Handler {
	server := NewServer(service, opts)

	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true, DisableLocalhostProtection: true})
}
