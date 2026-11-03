package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"

	sitedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type browserTestInput struct {
	Actions []sitedomain.BrowserAction `json:"actions"`
}

func browserTestDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: "browser_test", Description: "QA only: interact with the running project app using bounded browser actions.",
		Schema: json.RawMessage(`{"type":"object","properties":{"actions":{"type":"array","minItems":1,"maxItems":12,"items":{"type":"object","properties":{"action":{"type":"string","enum":["goto","click","fill","assert","screenshot"]},"path":{"type":"string"},"locator_type":{"type":"string","enum":["css","text","label","placeholder","role","testid"]},"locator":{"type":"string","maxLength":512},"value":{"type":"string","maxLength":512},"check":{"type":"string","enum":["visible","text"]}},"required":["action"]}}},"required":["actions"]}`),
	}
}

func browserScreenshotDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name:        "read_browser_screenshot",
		Description: "QA only: retrieve and inspect a screenshot from this task's browser_test result.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"screenshot_ref":{"type":"string","minLength":32,"maxLength":60}},"required":["screenshot_ref"]}`),
	}
}

func (a *agentTools) executeBrowserTest(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if scope.RoleKey != "qa" {
		return problem("Only the QA role can use the interactive browser.")
	}
	if a.sites == nil {
		return problem("Web apps are not available right now.")
	}
	if len(call.Input) > 24<<10 {
		return problem("The browser action list is too large; use at most 12 short actions.")
	}
	var input browserTestInput
	if err := json.Unmarshal(call.Input, &input); err != nil {
		return problem("The browser action list was not valid JSON.")
	}
	result, err := a.sites.BrowserForAgent(ctx, scope.UserID, scope.ProjectID, scope.TaskID, input.Actions)
	if err != nil {
		return problem(browserError(err).Error())
	}
	passed := true
	for _, step := range result.Steps {
		if step.Status != "PASS" {
			passed = false
		}
	}
	status := "passed"
	if !passed {
		status = "failed"
	}
	return taskapp.ToolResult{
		Output: fmt.Sprintf("Browser test %s. Screenshot attached to the task thread. Actions: %v", status, result.Steps),
		Image:  result.PNG, ScreenshotSiteVersion: result.SiteVersion,
	}
}

func (a *agentTools) readBrowserScreenshot(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if scope.RoleKey != "qa" || a.shots == nil || a.sites == nil || scope.TaskID == "" {
		return problem("Only the active QA role can read a screenshot from this task.")
	}
	var input struct {
		Ref string `json:"screenshot_ref"`
	}
	if err := json.Unmarshal(call.Input, &input); err != nil {
		return problem("That screenshot reference is not valid for this task.")
	}
	storedName, siteVersion, valid := taskdomain.ParseScreenshotReference(input.Ref, "")
	if !valid {
		return problem("That screenshot reference is not valid for this task.")
	}
	currentVersion, err := a.sites.RunningVersion(ctx, scope.UserID, scope.ProjectID)
	if err != nil || currentVersion != siteVersion {
		return problem("That screenshot expired when the app stopped or restarted; run browser_test again.")
	}
	data, err := a.shots.Load(ctx, scope.TaskID, storedName)
	if err != nil || len(data) == 0 || len(data) > 5<<20 {
		return problem("That screenshot is missing or expired; run browser_test again.")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return problem("The saved screenshot is not a valid PNG.")
	}
	return taskapp.ToolResult{
		Output: fmt.Sprintf("Retrieved screenshot_ref %s (image/png, %dx%d, %d bytes). Inspect the image pixels.", input.Ref, config.Width, config.Height, len(data)),
		Image:  data, ScreenshotSiteVersion: siteVersion,
	}
}
