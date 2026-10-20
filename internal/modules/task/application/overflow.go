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
	transcript, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil || !r.compact(ctx, task, c, clipResults(sinceSummary(transcript))) {
		return r.ask(ctx, client, request) // nothing shorter to send: let the model's own words be the error
	}

	reloaded, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil {
		return r.ask(ctx, client, request)
	}

	shortened := sinceSummary(reloaded)
	state.size = estimateTokens(c.system, c.tools, shortened)
	r.saveContext(ctx, task, state)

	if !state.noImages {
		shortened = r.withImages(ctx, task, shortened)
	}

	request.Messages = orderBlocks(shortened)

	return r.askSeeing(ctx, client, request, state)
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
