package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"

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
		Image:  result.PNG,
	}
}
