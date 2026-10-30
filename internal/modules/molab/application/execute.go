package application

import (
	"context"
	"regexp"
	"strings"

	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const inspectionCode = "import marimo._code_mode as cm; help(cm)"
const maxCodeBytes = 32_000
const maxOutputBytes = 12_000

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]{8,512}$`)

// Runner is the existing isolated one-shot Docker sandbox. It must not inherit
// Project Env or Git credentials and must add only the validated HTTPS hostname.
type Runner interface {
	RunMoLab(context.Context, codingdomain.Task, string, string) (codingapp.RunResult, error)
}

// Result is safe for MCP responses and logs. It never contains the bearer token.
type Result struct {
	Output  string
	Problem string
}

func Execute(ctx context.Context, runner Runner, rawURL, token, code string) Result {
	url, host, err := ValidateURL(rawURL)
	if err != nil {
		return Result{Problem: "Invalid MoLab URL. Use the notebook HTTPS address under sb-*.sb.molab.run."}
	}
	if !tokenPattern.MatchString(token) {
		return Result{Problem: "Invalid MoLab token format. Supply the temporary token from the active notebook."}
	}
	if strings.TrimSpace(code) == "" || len(code) > maxCodeBytes {
		return Result{Problem: "Provide Python code between 1 and 32000 bytes."}
	}
	if runner == nil {
		return Result{Problem: "The project sandbox is not available."}
	}

	// The official skill requires a successful inspection *before* user code,
	// not the two in a single scratchpad call.
	if result := run(ctx, runner, url, host, token, inspectionCode); result.Problem != "" {
		return result
	}
	return run(ctx, runner, url, host, token, code)
}

func run(ctx context.Context, runner Runner, url, host, token, code string) Result {
	task := codingdomain.Task{
		Name:           "marimo-pair",
		Command:        []string{"bash", "/usr/local/share/marimo-pair/scripts/execute-code.sh", "--url", url, "-c", code},
		TimeoutSeconds: 120,
	}
	out, err := runner.RunMoLab(ctx, task, token, host)
	if err != nil {
		return Result{Problem: "Could not start the isolated MoLab sandbox. Check Docker availability."}
	}

	safe := strings.ReplaceAll(out.Output, token, "[REDACTED]")
	if out.TimedOut {
		return Result{Problem: "MoLab did not respond before the timeout. Check the active notebook session."}
	}
	if out.ExitCode != 0 {
		return Result{Problem: classifyFailure(safe)}
	}
	if len(safe) > maxOutputBytes {
		safe = safe[:maxOutputBytes] + "\n[Output truncated]"
	}
	return Result{Output: safe}
}

func classifyFailure(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "401"), strings.Contains(lower, "403"), strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "forbidden"):
		return "MoLab token expired or was rejected. Send a fresh URL and token."
	case strings.Contains(lower, "no active sessions"), strings.Contains(lower, "no active session"):
		return "MoLab is running, but no notebook session is open. Open it and retry."
	case strings.Contains(lower, "failed to connect"), strings.Contains(lower, "could not reach"),
		strings.Contains(lower, "resolve"), strings.Contains(lower, "timed out"), strings.Contains(lower, "couldn't connect"):
		return "MoLab is unreachable or terminated. Start a new notebook and send its new URL and token."
	default:
		return "MoLab execution failed. Check the notebook Python output: " + bounded(output)
	}
}

func bounded(text string) string {
	if len(text) > maxOutputBytes {
		return text[:maxOutputBytes] + "\n[Output truncated]"
	}
	return text
}
