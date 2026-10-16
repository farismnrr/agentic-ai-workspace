package application

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Names of the tools that only exist on a team task.
const (
	toolDelegate = "delegate"
	toolReport   = "report"
)

const maxBrief = 4000

// delegateTool lets the orchestrator hand a job to one of the teammates.
func delegateTool(members []AgentRole) domain.ToolDef {
	keys, lines := make([]string, 0, len(members)), make([]string, 0, len(members))

	for _, member := range members {
		keys = append(keys, member.Key)
		lines = append(lines, member.Key+" ("+member.Name+"): "+member.Purpose)
	}

	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"role":    map[string]any{"type": "string", "enum": keys},
			"brief":   map[string]any{"type": "string", "description": "What the teammate must do and what done looks like."},
			"context": map[string]any{"type": "string", "description": "Facts, file names or earlier findings they need."},
		},
		"required": []string{"role", "brief"},
	})

	return domain.ToolDef{
		Name: toolDelegate,
		Description: "Hand one focused job to a teammate and wait for their report. The teammate sees only the brief " +
			"and context you write. Teammates:\n- " + strings.Join(lines, "\n- "),
		Schema: schema,
	}
}

// reportTool is how a teammate ends its job; it replaces finish for them.
func reportTool() domain.ToolDef {
	return domain.ToolDef{
		Name: toolReport,
		Description: "Call when your job is done. Say what you found or did, what you ran and how it ended, " +
			"and anything still open. When your job is to write a document such as the PRD, put it in document.",
		Schema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"},` +
			`"findings":{"type":"array","items":{"type":"string"}},` +
			`"document":{"type":"string","description":"A markdown document the person and the team will read."}},` +
			`"required":["summary"]}`),
	}
}

type delegateArgs struct {
	Role    string `json:"role"`
	Brief   string `json:"brief"`
	Context string `json:"context"`
}

func parseDelegate(input json.RawMessage) (delegateArgs, bool) {
	var args delegateArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return delegateArgs{}, false
	}

	args.Role, args.Brief = strings.TrimSpace(args.Role), strings.TrimSpace(args.Brief)

	return args, args.Role != "" && args.Brief != ""
}

// briefText is what the teammate reads first.
func (a delegateArgs) briefText() string {
	text := truncate(a.Brief, maxBrief)
	if context := strings.TrimSpace(a.Context); context != "" {
		text += "\n\nContext from the orchestrator:\n" + truncate(context, maxBrief)
	}

	return text
}

// parseReport turns a report call into the text the orchestrator reads and the document it came with, if any.
func parseReport(input json.RawMessage) (text, document string, ok bool) {
	var args struct {
		Summary  string   `json:"summary"`
		Document string   `json:"document"`
		Findings []string `json:"findings"`
	}

	if err := json.Unmarshal(input, &args); err != nil || strings.TrimSpace(args.Summary) == "" {
		return "", "", false
	}

	text = strings.TrimSpace(args.Summary)
	for _, finding := range args.Findings {
		text += "\n- " + strings.TrimSpace(finding)
	}

	return truncate(text, maxBrief), strings.TrimSpace(args.Document), true
}

// loopToolAllowed says whether this runner may use one of the loop's own tools: only the orchestrator
// plans, asks, finishes and delegates, and only a teammate reports.
func (r *Runner) loopToolAllowed(name string) bool {
	switch name {
	case toolUpdatePlan, toolFinish, toolDelegate:
		return r.role == nil
	case toolAskUser:
		return r.role == nil || r.canAsk()
	case toolReport:
		return r.role != nil
	}

	return true
}

// reportBack ends a teammate's job with their report.
func (r *Runner) reportBack(ctx context.Context, task domain.Task, call Call) (outcome, bool) {
	text, document, ok := parseReport(call.Input)
	if !ok {
		r.result(ctx, task, call.ID, "report needs a summary.", true)

		return outcome{}, false
	}

	r.result(ctx, task, call.ID, "Report sent.", false)

	return outcome{stop: stopFinished, text: text, document: document}, true
}

// canAsk reports whether this runner is a teammate allowed to ask the person questions, such as the Planner.
func (r *Runner) canAsk() bool {
	return r.role != nil && slices.Contains(r.role.Tools, toolAskUser)
}

// askTool is the loop's ask_user tool, for the teammates who may use it.
func askTool() domain.ToolDef {
	for _, tool := range agentTools() {
		if tool.Name == toolAskUser {
			return tool
		}
	}

	return domain.ToolDef{}
}
