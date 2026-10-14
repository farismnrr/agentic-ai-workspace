package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sitedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// The agent's tools for web apps. They need the project id, so they are handled apart from
// the file and command tools, and they are not offered to connected apps like ChatGPT.
const (
	toolStartSite = "start_site"
	toolStopSite  = "stop_site"
)

const startSiteHelp = "Start a web app in the sandbox so the person can open it in their browser. The command " +
	"must keep running and listen on 127.0.0.1 at the given port. The page is opened under a path, so the server " +
	"removes that path before the request reaches the app: use relative links (./style.css), or read BASE_PATH. " +
	"Use a production-style server, not a dev server with hot reload. Pages run in a sandbox: cookies, " +
	"localStorage and service workers do not work. Starting again replaces the running app."

func siteDefinitions() []taskdomain.ToolDef {
	return []taskdomain.ToolDef{
		{
			Name: toolStartSite, Description: startSiteHelp,
			Schema: json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":` +
				`"Shell command that starts the server and keeps running, such as python3 -m http.server 8000"},` +
				`"port":{"type":"integer","description":"The port it listens on, 1024 to 65535"}},` +
				`"required":["command","port"]}`),
		},
		{
			Name: toolStopSite, Description: "Stop the project's running web app.",
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
			"Say so in plain words; do not paste the address.",
		Link: link,
	}
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
