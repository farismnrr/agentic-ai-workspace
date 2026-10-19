package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	sitedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// The site tools need the project id, so they are handled apart from the file and command tools.
var (
	toolStartSite = codingdomain.ToolStartSite.Name
	toolStopSite  = codingdomain.ToolStopSite.Name
)

func siteDefinitions() []taskdomain.ToolDef {
	return []taskdomain.ToolDef{
		{
			Name: toolStartSite, Description: codingdomain.ToolStartSite.Description,
			Schema: json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":` +
				`"Shell command that starts the server and keeps running, such as python3 -m http.server 8000"},` +
				`"port":{"type":"integer","description":"The port it listens on, 1024 to 65535"}},` +
				`"required":["command","port"]}`),
		},
		{
			Name: toolStopSite, Description: codingdomain.ToolStopSite.Description,
			Schema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}
}

var siteDescribers = map[string]describer{
	toolStartSite: func(in toolArgs, _ string) taskapp.Described {
		return said(
			fmt.Sprintf("Started your app on port %d", in.Port),
			"May I start your app so you can open it in your browser?",
			fmt.Sprintf("start_site %s (port %d)", in.Command, in.Port))
	},
	toolStopSite: func(toolArgs, string) taskapp.Described {
		return said("Stopped your app", "May I stop your app?", "stop_site")
	},
}

func isSiteTool(name string) bool { return name == toolStartSite || name == toolStopSite }

func (a *agentTools) executeSite(ctx context.Context, projectID string, call taskapp.Call) taskapp.ToolResult {
	if a.sites == nil {
		return problem("Web apps are not available right now.")
	}

	if call.Name == toolStopSite {
		a.sites.StopProject(ctx, projectID)

		return taskapp.ToolResult{Output: "The app is stopped."}
	}

	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	site, err := a.sites.Start(ctx, projectID, in.Command, in.Port)
	if err != nil {
		return problem(siteFailure(err))
	}

	link := site.BasePath() + "/"

	return taskapp.ToolResult{
		Output: "The app is running and the person can open it from the link shown in the thread (" + link + "). " +
			"Say so in plain words; do not paste the address." + pathReport(a.sites.Audit(ctx, site)),
		Link: link,
	}
}

// pathReport tells the agent which links of the page would not work under the app's address, so it fixes them
// before saying the app is done. Nothing is added when every link is right.
func pathReport(problems []string) string {
	if len(problems) == 0 {
		return ""
	}

	return "\n\nBUT these links would not work when the person opens the app (it lives under an address prefix " +
		"that is removed from requests, so links must be relative):\n- " + strings.Join(problems, "\n- ") +
		"\nFix them, then call start_site again. Do not tell the person the app is ready until this list is empty."
}

// siteFailure words why a site did not start. A start that never answered carries the app's
// own output, which is what the model needs to fix it.
func siteFailure(err error) string {
	switch {
	case errors.Is(err, sitedomain.ErrNotStarted), errors.Is(err, sitedomain.ErrNoSandboxes):
		return truncateText(err.Error(), maxToolText)
	case errors.Is(err, sitedomain.ErrBadPort), errors.Is(err, sitedomain.ErrBadCommand),
		errors.Is(err, sitedomain.ErrTooMany), errors.Is(err, sitedomain.ErrServerBusy):
		return err.Error()
	}

	return "The app could not be started."
}

const maxToolText = 4000

func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}

	return text[len(text)-limit:]
}

// projectSites lets a connected app start and stop the web app of its project, the way the agent does.
type projectSites struct {
	tools     *agentTools
	projectID string
}

func (s projectSites) Start(ctx context.Context, command string, port int) codingmcp.SiteStart {
	if s.tools.sites == nil {
		return codingmcp.SiteStart{Problem: "Web apps are not available right now."}
	}

	site, err := s.tools.sites.Start(ctx, s.projectID, command, port)
	if err != nil {
		return codingmcp.SiteStart{Problem: siteFailure(err)}
	}

	return codingmcp.SiteStart{Link: s.tools.public + site.BasePath() + "/"}
}

func (s projectSites) Stop(ctx context.Context) {
	if s.tools.sites != nil {
		s.tools.sites.StopProject(ctx, s.projectID)
	}
}
