package application

import (
	"context"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// summaryResultChars is how much of each tool result a forced summary reads: what the model needs is the
// gist, and a request that already did not fit must not be asked again at the same size.
const summaryResultChars = 2_000

// overflowWords are how providers say that a request is longer than the model's window.
var overflowWords = []string{
	"context length", "context window", "context_length", "maximum context", "too many tokens",
	"prompt is too long", "input is too long", "exceeds the limit of", "reduce the length", "token limit",
}

// isOverflow reports whether the model refused the request for being too long.
func isOverflow(err error) bool {
	modelErr, ok := domain.AsModelError(err)
	if !ok || modelErr.Retryable {
		return false
	}

	text := strings.ToLower(modelErr.Message)

	return slices.ContainsFunc(overflowWords, func(word string) bool { return strings.Contains(text, word) })
}

// askShorter is for a request the model said was too long: the window was guessed too big, so the
// conversation is summarised now, from a trimmed copy, and the request is asked again once.
func (r *Runner) askShorter(
	ctx context.Context, task domain.Task, state *runState, c compaction, client Model, request Request,
) (Reply, error) {
	r.setCompactionState(ctx, task, "compacting")
	transcript, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	var compactInput []domain.Message
	if err == nil {
		compactInput = boundedSummaryInput(c.system, nil, sinceSummary(transcript), state.window)
	}
	if err != nil || len(compactInput) == 0 || !r.compact(ctx, task, c, compactInput) {
		r.setCompactionState(ctx, task, "failed")
		return Reply{}, &domain.ModelError{Message: "Context is full and automatic compaction failed. Retry after reducing the conversation or switching to a larger context window."}
	}

	reloaded, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil {
		r.setCompactionState(ctx, task, "failed")
		return Reply{}, &domain.ModelError{Message: "Context was compacted but could not reload the summary."}
	}

	shortened := sinceSummary(reloaded)
	state.size = estimateTokens(c.system, c.tools, shortened)
	if state.size >= compactThreshold(state.window) {
		r.setCompactionState(ctx, task, "failed")
		return Reply{}, &domain.ModelError{Message: "The compacted context is still too large for the selected model."}
	}
	r.saveContext(ctx, task, state)

	if !state.noImages {
		shortened = r.withImages(ctx, task, shortened)
	}

	request.Messages = orderBlocks(shortened)

	reply, askErr := r.askSeeing(ctx, client, request, state)
	if isOverflow(askErr) {
		r.setCompactionState(ctx, task, "failed")
	}
	return reply, askErr
}

// clipResults is a copy of the conversation whose tool results are cut short.
func clipResults(messages []domain.Message) []domain.Message {
	out := slices.Clone(messages)

	for i := range out {
		out[i].Blocks = slices.Clone(out[i].Blocks)

		for j := range out[i].Blocks {
			if out[i].Blocks[j].Kind == domain.BlockToolResult {
				out[i].Blocks[j].Text = truncate(out[i].Blocks[j].Text, summaryResultChars)
			}
		}
	}

	return out
}
