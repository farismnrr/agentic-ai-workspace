package mcp

import (
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Services are the use cases the tools call.
type Services struct {
	Reader *application.Service
	Editor *application.Editor
	Tasks  *application.Tasks
}

// Options connect the tools to permissions and the activity log.
type Options struct {
	Gate     Gate
	Recorder Recorder
	// ResourceMetadataURL is advertised in scope challenges.
	ResourceMetadataURL string
}

type tools struct {
	Services
	guard *guard
}

// NewServer builds the MCP server exposing the coding tools.
func NewServer(services Services, opts Options) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "masih-awam-coding", Version: "2.0.0"}, nil)
	g := &guard{gate: opts.Gate, recorder: opts.Recorder, metadataURL: opts.ResourceMetadataURL}
	t := &tools{Services: services, guard: g}

	t.addReadTools(server)
	t.addWriteTools(server)
	t.addExecTools(server)

	return server
}

// toolOf describes a tool to clients, including the OAuth scope it needs so
// ChatGPT can prompt for the right permission per tool.
func toolOf(info domain.ToolInfo) *sdk.Tool {
	closed := info.Group != domain.GroupExec

	return &sdk.Tool{
		Name: info.Name, Title: info.Title, Description: info.Description,
		Meta: sdk.Meta{"securitySchemes": []map[string]any{{"type": "oauth2", "scopes": []string{info.Scope}}}},
		Annotations: &sdk.ToolAnnotations{
			Title: info.Title, ReadOnlyHint: info.ReadOnly, OpenWorldHint: &closed,
		},
	}
}

// userFacing keeps domain errors readable for the model and hides everything else.
func userFacing(err error) error {
	for _, known := range knownErrors {
		if errors.Is(err, known) {
			return known
		}
	}

	return fmt.Errorf("the workspace could not be used: %w", errInternal)
}

var errInternal = errors.New("internal error")

var knownErrors = []error{
	domain.ErrDenied, domain.ErrNotFound, domain.ErrIsDir, domain.ErrNotDir, domain.ErrBinary, domain.ErrEmptyQuery,
	domain.ErrTooLarge, domain.ErrExists, domain.ErrNotUnique, domain.ErrNoMatch, domain.ErrBadRange,
	domain.ErrBadPattern, domain.ErrNoTasks, domain.ErrUnknownTask, domain.ErrBadTasks, domain.ErrTaskBusy,
}
