package application

import (
	"context"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	// compactReserve is room kept free in the window: 20k for the answer, 13k as a buffer.
	compactReserve  = 33_000
	maxCompactTries = 3
	charsPerToken   = 4
	minWindowShare  = 2 // a window is never considered full before half of it is used
	compactNote     = "I shortened my notes to keep going."
	summaryLead     = "Summary of the work so far (my own notes, written when the conversation got long):\n\n"
	summaryTail     = "\n\nContinue the task from here."
	compactAsk      = "The conversation has become long. Write a summary for yourself that lets you carry on " +
		"without the earlier messages. Keep the person's original request and latest instructions, decisions " +
		"made, files and commands you touched and what came of them, problems still open, and the next step. " +
		"Do not call tools; reply with the summary only."
)

// runState is what one run remembers between model turns.
type runState struct {
	// budget is the model turns left in the whole run; a teammate's loop shares the run's budget.
	budget *budget
	// roster is the team the orchestrator leads on this run; nil for a solo task and for teammates.
	roster *Roster
	// size is the tokens the last model call carried; zero while unknown.
	size int64
	// window is the model's context window in tokens, as of the last turn.
	window int64
	// failed counts compactions that produced no summary, so a stuck one is not retried forever.
	failed int
	// segment is the model turns this loop may still take; a teammate's has a share of its own.
	segment int
	// noImages is set once the model refused a request with screenshots; it then gets their text reports only.
	noImages bool
	// planWarned is set once the orchestrator was told its plan was out of date, so it is told only once.
	planWarned bool
}

// compaction is what summarising needs from the turn that asks for it.
type compaction struct {
	client Model
	system string
	tools  []domain.ToolDef
}

// compactThreshold is the context size at which a window is considered full.
func compactThreshold(window int64) int64 {
	return max(window-compactReserve, window/minWindowShare)
}

// compactIfNeeded replaces a long conversation by a summary written by the model
// itself, and returns the messages to send. It never blocks the run: when no
// summary comes back, the conversation goes on as it is.
func (r *Runner) compactIfNeeded(
	ctx context.Context, task domain.Task, state *runState, c compaction, transcript []domain.Message,
) []domain.Message {
	messages := sinceSummary(transcript)

	size := state.size
	if size == 0 {
		size = estimateTokens(c.system, c.tools, messages)
	}

	window := r.deps.Windows.ContextWindow(ctx, task.UserID, task.ProviderID, task.Model)
	state.window = window

	if size < compactThreshold(window) || state.failed >= maxCompactTries {
		return messages
	}

	if !r.compact(ctx, task, c, messages) {
		state.failed++

		return messages
	}

	reloaded, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil {
		state.size = 0

		return messages
	}

	shortened := sinceSummary(reloaded)
	state.size = estimateTokens(c.system, c.tools, shortened)
	r.saveContext(ctx, task, state)

	return shortened
}

// saveContext tells the task page how full the model's window is.
func (r *Runner) saveContext(ctx context.Context, task domain.Task, state *runState) {
	if r.role != nil {
		return // the meter follows the orchestrator
	}

	_ = r.deps.Tasks.SetContext(ctx, task.ID, state.size, state.window)
}

// compact asks the model for a summary and records it; false means none came back.
func (r *Runner) compact(ctx context.Context, task domain.Task, c compaction, messages []domain.Message) bool {
	reply, err := r.ask(ctx, c.client, Request{
		Model: task.Model, System: c.system, Effort: task.Effort, Messages: orderBlocks(withAsk(messages)), Tools: c.tools,
	})
	if err != nil {
		return false
	}

	r.recordUsage(ctx, task, reply.Usage)

	text, ok := summaryText(reply)
	if !ok {
		return false
	}

	_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{
		Kind: domain.BlockText, Text: summaryLead + text + summaryTail, Summary: true,
	})
	r.say(ctx, task.ID, compactNote)

	return true
}

// summaryText is the summary in a reply; none when the model reached for a tool or wrote nothing.
func summaryText(reply Reply) (string, bool) {
	texts := []string{}

	for _, block := range reply.Blocks {
		switch block.Kind {
		case domain.BlockToolUse:
			return "", false
		case domain.BlockText:
			texts = append(texts, strings.TrimSpace(block.Text))
		case domain.BlockToolResult:
		}
	}

	text := strings.TrimSpace(strings.Join(texts, "\n\n"))

	return text, text != ""
}

// withAsk adds the request for a summary as the last thing the person "says".
func withAsk(messages []domain.Message) []domain.Message {
	ask := domain.Block{Kind: domain.BlockText, Text: compactAsk}
	out := slices.Clone(messages)

	if last := len(out) - 1; last >= 0 && out[last].Role == domain.RoleUser {
		out[last].Blocks = append(slices.Clone(out[last].Blocks), ask)

		return out
	}

	return append(out, domain.Message{Role: domain.RoleUser, Blocks: []domain.Block{ask}})
}

// sinceSummary is the conversation as the model sees it: from the latest summary on.
func sinceSummary(transcript []domain.Message) []domain.Message {
	for i := len(transcript) - 1; i >= 0; i-- {
		blocks := transcript[i].Blocks

		for j := len(blocks) - 1; j >= 0; j-- {
			if blocks[j].Summary {
				head := domain.Message{Role: transcript[i].Role, Blocks: blocks[j:]}

				return append([]domain.Message{head}, transcript[i+1:]...)
			}
		}
	}

	return transcript
}

// estimateTokens is a rough size of a request, for when no model call has measured it yet.
func estimateTokens(system string, tools []domain.ToolDef, messages []domain.Message) int64 {
	chars := len(system)

	for _, tool := range tools {
		chars += len(tool.Name) + len(tool.Description) + len(tool.Schema)
	}

	for _, message := range messages {
		for _, block := range message.Blocks {
			chars += len(block.Text) + len(block.Name) + len(block.Input)
		}
	}

	return int64(chars / charsPerToken)
}
