package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// SiteRunner starts and stops the web app of one project.
type SiteRunner interface {
	// Start runs the command. A start that failed returns the reason in Problem and no Link.
	Start(ctx context.Context, command string, port int) SiteStart
	Stop(ctx context.Context)
}

// BrowserRunner performs a QA-only browser run against one project Site and work session.
type BrowserRunner interface {
	Browser(ctx context.Context, userID, taskID string, actions []BrowserAction) (BrowserReport, error)
}

// SiteStart is how a start went.
type SiteStart struct {
	Link    string
	Problem string
}

type startSiteInput struct {
	Target
	Command string `json:"command" jsonschema:"shell command that starts the server and keeps running"`
	Port    int    `json:"port" jsonschema:"the port it listens on, 1024 to 65535"`
}

type siteOutput struct {
	URL     string `json:"url,omitempty" jsonschema:"address the person can open in their browser"`
	Message string `json:"message"`
	Running bool   `json:"running"`
}

func (t *tools) addSiteTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolStartSite), guarded(t.guard, domain.ToolStartSite,
		func(in startSiteInput) string { return "start web app: " + in.Command }, t.startSite))
	sdk.AddTool(server, toolOf(domain.ToolStopSite), guarded(t.guard, domain.ToolStopSite,
		func(Target) string { return "stop web app" }, t.stopSite))
	sdk.AddTool(server, toolOf(domain.ToolBrowserTest), guarded(t.guard, domain.ToolBrowserTest,
		func(in browserInput) string {
			return "test the web app with " + fmt.Sprint(len(in.Actions)) + " browser actions"
		},
		t.browserTest))
}

type browserInput struct {
	Target
	Actions []BrowserAction `json:"actions" jsonschema:"up to 12 supported actions: goto, click, fill, assert or screenshot"`
}

type browserOutput struct {
	Steps         []BrowserStep `json:"steps"`
	ScreenshotPNG string        `json:"screenshot_png_base64"`
}

func (t *tools) browserTest(ctx context.Context, in browserInput) (browserOutput, report, error) {
	if t.works == nil {
		return browserOutput{}, report{}, taskdomain.ErrUnknownWork
	}
	svc, err := t.services(ctx)
	if err != nil {
		return browserOutput{}, report{}, err
	}
	who, _ := ctx.Value(identityKey{}).(identity)
	if err = t.works.RequireRole(ctx, who.User, who.Client, svc.ProjectID, in.WorkID, "qa"); err != nil {
		return browserOutput{}, report{}, workToolError("require QA role", err)
	}
	state, err := t.works.State(ctx, who.User, who.Client, svc.ProjectID, in.WorkID)
	if err != nil || state.TaskID == "" {
		return browserOutput{}, report{}, workToolError("resolve active work task", taskdomain.ErrUnknownWork)
	}
	runner, ok := svc.Sites.(BrowserRunner)
	if !ok {
		return browserOutput{}, report{}, domain.ErrBrowserUnavailable
	}
	result, err := runner.Browser(ctx, who.User, state.TaskID, in.Actions)
	if err != nil {
		return browserOutput{}, report{}, err
	}
	return browserOutput{Steps: result.Steps, ScreenshotPNG: base64.StdEncoding.EncodeToString(result.PNG)},
		report{detail: "browser actions completed"}, nil
}

func (t *tools) startSite(ctx context.Context, in startSiteInput) (siteOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return siteOutput{}, report{}, err
	}

	started := svc.Sites.Start(ctx, in.Command, in.Port)
	if started.Link == "" {
		return siteOutput{Message: started.Problem}, report{detail: started.Problem, failed: true}, nil
	}

	message := "The app is running. Tell the person to open " + started.Link + " in their browser."

	return siteOutput{Running: true, URL: started.Link, Message: message}, report{detail: "running"}, nil
}

func (t *tools) stopSite(ctx context.Context, _ Target) (siteOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return siteOutput{}, report{}, err
	}

	svc.Sites.Stop(ctx)

	return siteOutput{Message: "The app is stopped."}, report{detail: "stopped"}, nil
}
