package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Services are the use cases the tools call.
type Services struct {
	Reader *application.Service
	Editor *application.Editor
	Tasks  *application.Tasks
	// Sites starts and stops the web app of this workspace's project.
	Sites SiteRunner
	// ProjectID is the project this workspace is.
	ProjectID string
}

// Options connect the tools to permissions and the activity log.
type Options struct {
	Gate     Gate
	Recorder Recorder
	// Tasks, when set, keeps what the app does as a thread the person can read.
	Tasks TaskLog
	// Works backs the explicit connected-app work lifecycle tools.
	Works WorkSessions
	// ResourceMetadataURL is advertised in scope challenges.
	ResourceMetadataURL string
}

// Workspaces gives each connected app a project of the person's own to work in. A call can only ever
// reach the project of the person whose token it carries, named after the app that made it.
type Workspaces interface {
	// For returns the services of the project named by its Project ID, or of the app's own project when project is
	// empty. It fails with domain.ErrUnknownProject for an ID that is none of the person's projects.
	For(ctx context.Context, userID, client, project string) (Services, error)
	// Projects lists the person's projects, marking the one that is the app's own.
	Projects(ctx context.Context, userID, client string) ([]ProjectInfo, error)
}

type tools struct {
	workspaces Workspaces
	guard      *guard
	works      WorkSessions
}

// services returns the workspace of the person whose token made this call.
func (t *tools) services(ctx context.Context) (Services, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	if who.User == "" {
		return Services{}, errNoIdentity
	}

	services, err := t.workspaces.For(ctx, who.User, who.Client, who.Project)
	if err != nil {
		return Services{}, fmt.Errorf("open workspace: %w", err)
	}
	if who.WorkID != "" {
		if t.works == nil {
			return Services{}, taskdomain.ErrUnknownWork
		}
		if err := t.works.Validate(ctx, who.User, who.Client, services.ProjectID, who.WorkID); err != nil {
			return Services{}, workToolError("validate work session", err)
		}
	}

	if state, ok := ctx.Value(stateKey{}).(*callState); ok {
		state.project = services.ProjectID

		if state.begin != nil && !state.started {
			state.started = true
			state.begin(services.ProjectID)
		}
	}

	return services, nil
}

// serverInstructions are shown to the app's model when it connects. The person follows the work in their own
// workspace, and only what the model writes down tells them what they asked for.
const serverInstructions = "These tools work in the person's own workspace, where they follow your work in a thread. " +
	"Start each request by calling log_message with role user and what the person asked, in their words. " +
	"End it by calling log_message with role assistant and a short summary of the result. " +
	"When the person names a project (such as sensio), call list_projects and give its id as project to the other " +
	"tools; without a project you work in your own. For substantial requests, call begin_work first (small tasks may " +
	"skip it), switch role labels as useful (Researcher, Planner, Developer, QA, Security), pass work_id to each " +
	"work tool, and finish with complete_work. There is one assistant: roles are labels, no other agent is called, " +
	"and permissions do not change. Start a web app with start_site, never with a shell command."

// NewServer builds the MCP server exposing the coding tools.
func NewServer(workspaces Workspaces, opts Options) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "masih-awam-coding", Version: "2.0.0"},
		&sdk.ServerOptions{Instructions: serverInstructions})
	g := &guard{gate: opts.Gate, recorder: opts.Recorder, tasks: opts.Tasks, metadataURL: opts.ResourceMetadataURL}
	t := &tools{workspaces: workspaces, guard: g, works: opts.Works}

	t.addReadTools(server)
	t.addWriteTools(server)
	t.addExecTools(server)
	t.addLogTools(server)
	t.addSiteTools(server)
	t.addProjectTools(server)
	t.addWorkTools(server)

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

// UserFacing keeps domain errors readable for the model and hides everything else.
func UserFacing(err error) error {
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
	domain.ErrServerBusy, domain.ErrUnknownProject, errBadMessage,
	taskdomain.ErrUnknownWork, taskdomain.ErrWorkClosed, taskdomain.ErrStaleRevision, taskdomain.ErrKeyReused,
	taskdomain.ErrBadRole, taskdomain.ErrTooMany, taskdomain.ErrRateLimited, taskdomain.ErrTooLarge,
	taskdomain.ErrBadRequest, taskdomain.ErrNoProject,
}
