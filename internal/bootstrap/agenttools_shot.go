package bootstrap

import (
	"context"
	"encoding/json"
	"errors"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	sitedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var toolScreenshot = codingdomain.ToolScreenshot.Name

func screenshotDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: toolScreenshot, Description: codingdomain.ToolScreenshot.Description,
		Schema: json.RawMessage(`{"type":"object","properties":{` +
			`"path":{"type":"string","description":"The page, such as /pricing. Default /"},` +
			`"device":{"type":"string","enum":["desktop","mobile"],` +
			`"description":"Default desktop (1280 wide); mobile is 390 wide"},` +
			`"y":{"type":"integer","description":"Pixels down the page to start at, for a closer look at a long page"},` +
			`"overview":{"type":"boolean","description":"The whole page scaled down instead of one screen"}}}`),
	}
}

func describeShot(in toolArgs) taskapp.Described {
	text := "Looked at your app on a desktop screen"
	if in.Device == "mobile" {
		text = "Looked at your app on a phone screen"
	}

	return said(text, "", "screenshot "+in.Path)
}

// executeShot photographs a page of the project's running app.
func (a *agentTools) executeShot(ctx context.Context, projectID string, call taskapp.Call) taskapp.ToolResult {
	if a.sites == nil {
		return problem("Web apps are not available right now.")
	}

	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	spec := sitedomain.CaptureSpec{Path: in.Path, Y: in.Y, Overview: in.Overview, Mobile: in.Device == "mobile"}

	shot, err := a.sites.Capture(ctx, projectID, spec)
	if err != nil {
		return problem(shotFailure(err))
	}

	spec, _ = spec.Normalize()

	return taskapp.ToolResult{Output: shot.Describe(spec), Image: shot.PNG}
}

func shotFailure(err error) string {
	switch {
	case errors.Is(err, sitedomain.ErrNotRunning), errors.Is(err, sitedomain.ErrBadPath),
		errors.Is(err, sitedomain.ErrNoSandboxes), errors.Is(err, sitedomain.ErrServerBusy):
		return err.Error()
	case errors.Is(err, sitedomain.ErrNoPicture):
		return truncateText(err.Error(), maxToolText)
	}

	return "The screenshot could not be taken."
}
