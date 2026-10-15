package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// SiteRunner starts and stops the web app of one project.
type SiteRunner interface {
	// Start runs the command. A start that failed returns the reason in Problem and no Link.
	Start(ctx context.Context, command string, port int) SiteStart
	Stop(ctx context.Context)
}

// SiteStart is how a start went.
type SiteStart struct {
	Link    string
	Problem string
}

type startSiteInput struct {
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
		func(struct{}) string { return "stop web app" }, t.stopSite))
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

func (t *tools) stopSite(ctx context.Context, _ struct{}) (siteOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return siteOutput{}, report{}, err
	}

	svc.Sites.Stop(ctx)

	return siteOutput{Message: "The app is stopped."}, report{detail: "stopped"}, nil
}
