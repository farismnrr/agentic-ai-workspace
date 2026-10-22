package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	siteapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	sitedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var toolScreenshotSession = codingdomain.ToolScreenshotSession.Name

func screenshotSessionDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: toolScreenshotSession,
		Description: codingdomain.ToolScreenshotSession.Description +
			" Login uses secret Project Env names and CSS selectors.",
		Schema: json.RawMessage(`{"type":"object","properties":{
"action":{"type":"string","enum":["login","logout","reset"]},
"path":{"type":"string","description":"App page path, such as /login"},
"email_secret":{"type":"string","description":"Project Env secret name for the test email"},
"password_secret":{"type":"string","description":"Project Env secret name for the test password"},
"email_selector":{"type":"string"},"password_selector":{"type":"string"},
"submit_selector":{"type":"string"},
"ready_selector":{"type":"string","description":"Selector that proves the requested page is ready"},
"error_selector":{"type":"string","description":"Optional visible login error selector"}},"required":["action"]}`),
	}
}

func (a *agentTools) executeScreenshotSession(
	ctx context.Context, scope taskapp.ToolScope, call taskapp.Call,
) taskapp.ToolResult {
	if a.sites == nil {
		return problem("Web apps are not available right now.")
	}
	var in toolArgs
	if json.Unmarshal(call.Input, &in) != nil {
		return problem("The session action was not valid JSON.")
	}
	action, failure := a.resolveSessionAction(ctx, scope, in)
	if failure != "" {
		return problem(failure)
	}
	result, err := a.sites.RunSessionAction(ctx, scope.UserID, scope.ProjectID, scope.TaskID, action)
	return sessionActionResult(in.Action, result, err)
}

func (a *agentTools) resolveSessionAction(
	ctx context.Context, scope taskapp.ToolScope, in toolArgs,
) (siteapp.SessionAction, string) {
	action := siteapp.SessionAction{
		Action: in.Action, Path: in.Path, EmailSelector: in.EmailSelector, PasswordSelector: in.PasswordSelector,
		SubmitSelector: in.SubmitSelector, ReadySelector: in.ReadySelector, ErrorSelector: in.ErrorSelector,
	}
	if in.Action == "login" {
		if a.secrets == nil {
			return action, "Project Env secrets are not available."
		}
		var err error
		action.Email, err = a.secrets.Secret(ctx, scope.UserID, scope.ProjectID, in.EmailSecret)
		if err != nil {
			return action, "The named email secret was not found in this project."
		}
		action.Password, err = a.secrets.Secret(ctx, scope.UserID, scope.ProjectID, in.PasswordSecret)
		if err != nil {
			return action, "The named password secret was not found in this project."
		}
	}
	return action, ""
}

func sessionActionResult(action string, result siteapp.SessionResult, err error) taskapp.ToolResult {
	if err != nil {
		if action == "logout" {
			return taskapp.ToolResult{Output: sessionStatusMessage(result.Status)}
		}
		if errors.Is(err, sitedomain.ErrNotRunning) {
			return problem("The project app is not running. Start it before changing its session.")
		}
		return problem("The session action could not be completed. Check the app, path, and selectors.")
	}
	return taskapp.ToolResult{Output: sessionStatusMessage(result.Status)}
}

func sessionStatusMessage(status string) string {
	if message, ok := sessionMessages[status]; ok {
		return message
	}
	return "The session action did not finish. No credentials or page errors were returned."
}

var sessionMessages = map[string]string{
	"local_session_cleared": "The agent's local session was cleared. The app's server session was not revoked.",
	"logout_submitted_ready_local_session_cleared": "The logout readiness indicator appeared and the local session " +
		"was cleared. Server revocation was not confirmed.",
	"logout_submitted_indicator_missing_local_session_cleared": "Logout was submitted and the local session cleared, " +
		"but the readiness indicator did not appear. Server revocation was not confirmed.",
	"logout_failed_local_session_cleared": "Logout failed, but the local session was cleared. The app's server " +
		"session may still exist.",
	"logout_timed_out_local_session_cleared": "Logout timed out, but the local session was cleared. The app's " +
		"server session may still exist.",
	"indicator_found": "The requested readiness indicator appeared. It does not prove the account's permissions.",
	"indicator_not_found": "The form was submitted, but the requested readiness indicator did not appear. " +
		"Login was not confirmed.",
	"login_failed":              "The app displayed its configured login error. Login failed.",
	"not_found":                 "A requested form control was not found on the page.",
	"external_redirect_blocked": "The app attempted to leave its own address. The external redirect was blocked.",
	"operation_timed_out":       "The browser action timed out. Login or logout was not confirmed.",
	"unsupported":               "This app's form did not match the supported email and password controls.",
}

func describeSessionAction(in toolArgs) taskapp.Described {
	action := in.Action
	if action == "" {
		action = "unknown"
	}
	target := sitedomain.CaptureReportPath(in.Path)
	if action == "reset" {
		target = "the local agent session"
	}
	selectors := sessionSelectors(in)
	text := fmt.Sprintf("%s the agent session for %s", actionLabel(action), target)
	if selectors != "" {
		text += " (" + selectors + ")"
	}
	ask := fmt.Sprintf("May I %s the agent session for %s", actionLabel(action), target)
	if selectors != "" {
		ask += " using " + selectors
	}
	return said(text, ask+"?", "screenshot_session "+action)
}

func actionLabel(action string) string {
	switch action {
	case "login":
		return "log in"
	case "logout":
		return "log out"
	case "reset":
		return "reset"
	default:
		return "change"
	}
}

func sessionSelectors(in toolArgs) string {
	values := []string{}
	for _, field := range []struct{ name, value string }{
		{"email", in.EmailSelector},
		{"password", in.PasswordSelector},
		{"submit", in.SubmitSelector},
		{"ready", in.ReadySelector},
		{"error", in.ErrorSelector},
	} {
		if value := strings.TrimSpace(field.value); value != "" {
			values = append(values, field.name+"="+sanitizeApprovalSelector(value))
		}
	}
	return strings.Join(values, ", ")
}

var approvalSecretAttribute = regexp.MustCompile(
	`(?i)(value|email|password|token|secret|csrf)(\s*=\s*)("[^"]*"|'[^']*')`,
)

const maxApprovalSelectorBytes = 120

func sanitizeApprovalSelector(value string) string {
	value = approvalSecretAttribute.ReplaceAllString(value, "$1$2[redacted]")
	if len(value) > maxApprovalSelectorBytes {
		return value[:maxApprovalSelectorBytes-3] + "..."
	}
	return value
}
