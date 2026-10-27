package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	githubdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	defaultTail = 120
	maxTail     = 400
	maxLogBytes = 24 << 10
)

var toolCILogs = codingdomain.ToolCILogs.Name

type ciLogsArgs struct {
	Repo     string `json:"repo"`
	RunID    int64  `json:"run_id"`
	JobID    int64  `json:"job_id"`
	TailLine int    `json:"tail_lines"`
}

func ciLogsDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: toolCILogs, Description: codingdomain.ToolCILogs.Description,
		Schema: schema(`"repo":{"type":"string","description":"The repository as owner/name"},`+
			`"run_id":{"type":"integer","description":"The run number; gives the logs of its failed jobs"},`+
			`"job_id":{"type":"integer","description":"One job number, instead of a run"},`+
			`"tail_lines":{"type":"integer","description":"How many last lines of each log (default 120, at most 400)"}`,
			`"repo"`),
	}
}

func describeCILogs(toolArgs) taskapp.Described {
	return said("Read why a CI run failed", "May I read the logs of a CI run?", toolCILogs)
}

// executeCILogs reads the logs as the person who owns the project, never as whoever asks: the owner is looked up
// from the project, the same rule that signs commands in to GitHub.
func (a *agentTools) executeCILogs(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	var in ciLogsArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	if a.github == nil || a.owners == nil {
		return problem("Reading CI logs is not available right now.")
	}

	owner, _, err := a.owners.Owner(ctx, scope.ProjectID)
	if err != nil {
		return problem("This project has no owner to read CI logs as.")
	}

	logs, err := a.github.CILogs(ctx, owner, githubdomain.LogRef{Repo: in.Repo, RunID: in.RunID, JobID: in.JobID})
	if err != nil {
		return problem(ciLogsFailure(err))
	}

	return taskapp.ToolResult{Output: formatCILogs(logs, clampTail(in.TailLine))}
}

func clampTail(lines int) int {
	if lines <= 0 {
		return defaultTail
	}

	return min(lines, maxTail)
}

func ciLogsFailure(err error) string {
	switch {
	case errors.Is(err, githubdomain.ErrBadLogRef):
		return err.Error() + "."
	case errors.Is(err, githubdomain.ErrNotConnected), errors.Is(err, githubdomain.ErrNotConfigured):
		return "The project's owner has not connected GitHub, so CI logs cannot be read. Tell the person to connect " +
			"it in Settings."
	case errors.Is(err, githubdomain.ErrLogsDenied):
		return "GitHub did not allow reading these logs: the app may lack the Actions read permission, or this " +
			"repository was not shared with it. Report exactly that; do not ask the person to change a token scope."
	case errors.Is(err, githubdomain.ErrLogsGone):
		return "There are no such logs: check the repository and the run or job number, and note that GitHub deletes " +
			"logs after 90 days."
	}

	return "GitHub could not be reached right now. Try once more, then report it as Stuck."
}

func formatCILogs(logs []githubdomain.JobLog, tail int) string {
	if len(logs) == 0 {
		return "No job of that run failed, so there is no failure log. Its jobs may still be running, or it passed."
	}

	var out strings.Builder

	for _, log := range logs {
		title := log.Name
		if title == "" {
			title = "job"
		}

		fmt.Fprintf(&out, "== %s %s (last %d lines) ==\n%s\n\n", title, log.Conclusion, tail,
			githubdomain.Tail(log.Text, tail, maxLogBytes))
	}

	return strings.TrimSpace(out.String())
}
