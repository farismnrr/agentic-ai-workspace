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

// compactIfNeeded replaces a long conversation with a bounded summary.
// On compaction failure it stops the turn; never retry an oversized prompt unchanged.
func (r *Runner) compactIfNeeded(
	ctx context.Context, task domain.Task, state *runState, c compaction, transcript []domain.Message,
) ([]domain.Message, bool) {
	messages := sinceSummary(transcript)

	size := state.size
	// A previous call's usage does not include newly appended tool results or user input.
	size = max(size, estimateTokens(c.system, c.tools, messages))

	window := r.deps.Windows.ContextWindow(ctx, task.UserID, task.ProviderID, task.Model)
	state.window = window

	if size < compactThreshold(window) {
		return messages, true
	}
	state.size = size
	r.setCompactionState(ctx, task, "compacting")
	if state.failed >= maxCompactTries {
		r.setCompactionState(ctx, task, "failed")
		return nil, false
	}

	// A full transcript cannot reliably be used to ask for its own summary.
	// Bound the entire summary request, not just tool results. A giant user
	// message or tool argument can otherwise overflow the summary call itself.
	compactInput := boundedSummaryInput(c.system, nil, messages, window)
	if len(compactInput) == 0 {
		state.failed++
		r.setCompactionState(ctx, task, "failed")
		return nil, false
	}
	if !r.compact(ctx, task, c, compactInput) {
		state.failed++
		r.setCompactionState(ctx, task, "failed")
		return nil, false
	}

	reloaded, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil {
		state.size = 0
		r.setCompactionState(ctx, task, "failed")
		return nil, false
	}

	shortened := sinceSummary(reloaded)
	state.size = estimateTokens(c.system, c.tools, shortened)
	if state.size >= compactThreshold(window) {
		r.setCompactionState(ctx, task, "failed")
		return nil, false // model returned an unusably large summary
	}
	r.saveContext(ctx, task, state)

	return shortened, true
}

// saveContext tells the task page how full the model's window is.
func (r *Runner) saveContext(ctx context.Context, task domain.Task, state *runState) {
	if r.role != nil {
		return // the meter follows the orchestrator
	}

	_ = r.deps.Tasks.SetContext(ctx, task.ID, state.size, state.window)
}

func (r *Runner) setCompactionState(ctx context.Context, task domain.Task, status string) {
	if r.role == nil {
		_ = r.deps.Tasks.SetContextState(ctx, task.ID, status)
	}
}

// compact asks the model for a summary and records it; false means none came back.
func (r *Runner) compact(ctx context.Context, task domain.Task, c compaction, messages []domain.Message) bool {
	reply, err := r.ask(ctx, c.client, Request{
		Model: task.Model, System: c.system, Effort: task.Effort, Messages: orderBlocks(withAsk(messages)),
		// Never offer executable tools during summary-only compaction.
	})
	if err != nil {
		return false
	}

	r.recordUsage(ctx, task, reply.Usage)

	text, ok := summaryText(reply)
	if !ok {
		return false
	}
	// Avoid persisting a giant summary that immediately overflows again.
	window := r.deps.Windows.ContextWindow(ctx, task.UserID, task.ProviderID, task.Model)
	maxSummaryTokens := compactThreshold(window) - estimateTokens(c.system, nil, nil) - 500
	if maxSummaryTokens <= 0 || int64(len(text)) > maxSummaryTokens*charsPerToken {
		return false
	}

	if err := r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, domain.Block{
		Kind: domain.BlockText, Text: summaryLead + text + summaryTail, Summary: true,
	}); err != nil {
		return false
	}
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


