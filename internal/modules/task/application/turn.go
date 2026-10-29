package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxModelError = 300

// turn sends the conversation to the model and records its answer. It reports
// done when the run is over, which happens when the model stops asking for tools.
func (r *Runner) turn(
	ctx context.Context, task domain.Task, state *runState, transcript []domain.Message,
) (outcome, bool) {
	client, err := r.deps.Models.Client(ctx, task.UserID, task.ProviderID)
	if err != nil {
		return failed("I cannot reach the AI model you chose.", "Check Settings, AI models."), true
	}

	offered, err := r.offered(ctx, task.UserID, state)
	if err != nil {
		return failed("I cannot check which tools you switched on.", "Try Continue in a moment."), true
	}

	system := r.system(ctx, task, state)
	shrink := compaction{client: client, system: system, tools: offered}
	messages, ready := r.compactIfNeeded(ctx, task, state, shrink, transcript)
	if !ready {
		return failed("Context is full and automatic compaction did not succeed.", "Retry, reduce the conversation, or choose a model with a larger context window."), true
	}

	if !state.noImages {
		messages = r.withImages(ctx, task, messages)
	}

	request := Request{
		Model: task.Model, System: system, Effort: task.Effort, Messages: orderBlocks(messages), OnText: r.streamTo(task.ID),
		Tools: offered,
	}

	reply, err := r.askSeeing(ctx, client, request, state)
	if isOverflow(err) {
		reply, err = r.askShorter(ctx, task, state, shrink, client, request)
	}

	if err != nil {
		return r.modelFailure(ctx, err), true
	}

	r.recordUsage(ctx, task, reply.Usage)
	state.size = reply.Usage.Input + reply.Usage.Output
	r.saveContext(ctx, task, state)

	return r.record(ctx, task, state, reply)
}

func (r *Runner) recordUsage(ctx context.Context, task domain.Task, usage domain.Usage) {
	_ = r.deps.Usage.AddUsage(ctx, UsageRecord{
		At: r.deps.Now(), TaskID: task.ID, UserID: task.UserID, ProviderID: task.ProviderID, Model: task.Model,
		Usage: usage,
	})
}

func (r *Runner) record(ctx context.Context, task domain.Task, state *runState, reply Reply) (outcome, bool) {
	if len(reply.Blocks) == 0 {
		return failed("The AI model sent an empty answer.", "Try Continue, or pick another model."), true
	}

	calls, say := r.recordBlocks(ctx, task.ID, reply.Blocks)
	if !calls && r.role != nil {
		// A teammate that stops asking for tools has said what it has to say: that is its report.
		return outcome{stop: stopFinished, text: say}, true
	}

	if calls {
		r.say(ctx, task.ID, say)

		return outcome{}, false
	}

	// No tool calls: the model considers itself done.
	events, err := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return failed("I lost track of this task's history.", "Try Continue in a moment."), true
	}
	if !workDone(events) {
		return r.answer(ctx, task, say), true
	}

	blocked, allowed := r.completionGate(ctx, task, state, Call{})
	if !allowed {
		if blocked.stop != 0 {
			return blocked, true
		}
		return outcome{}, false
	}

	return r.complete(ctx, task, say, buildProof(events, nil)), true
}

func (r *Runner) recordBlocks(ctx context.Context, taskID string, blocks []domain.Block) (bool, string) {
	calls, texts := false, []string{}
	for _, block := range blocks {
		_ = r.deps.Transcript.AppendBlock(ctx, taskID, domain.RoleAssistant, block)
		if block.Kind == domain.BlockToolUse {
			calls = true
		}
		if block.Kind == domain.BlockText {
			texts = append(texts, strings.TrimSpace(block.Text))
		}
	}
	return calls, strings.TrimSpace(strings.Join(texts, "\n\n"))
}

func (r *Runner) say(ctx context.Context, taskID, text string) {
	if text == "" {
		return
	}

	_, _ = r.deps.Events.AppendEvent(ctx, taskID, domain.Event{Kind: domain.EventMessage, Text: text, Time: r.deps.Now()})
}

// ask calls the model, retrying rate limits and outages with a pause.
func (r *Runner) ask(ctx context.Context, client Model, request Request) (Reply, error) {
	for attempt := 0; ; attempt++ {
		reply, err := client.Complete(ctx, request)
		if err == nil {
			return reply, nil
		}

		modelErr, ok := domain.AsModelError(err)
		if !ok || !modelErr.Retryable || attempt >= r.lim.Retries {
			return Reply{}, fmt.Errorf("model call: %w", err)
		}

		select {
		case <-time.After(r.lim.Backoff(attempt)):
		case <-ctx.Done():
			return Reply{}, ctx.Err() //nolint:wrapcheck // the caller only checks the context
		}
	}
}

func (r *Runner) modelFailure(ctx context.Context, err error) outcome {
	if ctx.Err() != nil {
		result, _ := r.limitReached(ctx)

		return result
	}

	message := "The AI model could not answer."
	if modelErr, ok := domain.AsModelError(err); ok {
		message = "The AI model said: " + truncate(modelErr.Message, maxModelError)
	}

	return failed(message, "Check your key and model under Settings, AI models, then press Continue.")
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return string(runes[:limit]) + "…"
}

// orderBlocks puts tool results ahead of text inside each user message, which
// every provider requires when a person writes while calls are still unanswered.
func orderBlocks(messages []domain.Message) []domain.Message {
	ordered := make([]domain.Message, 0, len(messages))

	for _, message := range messages {
		results, rest := []domain.Block{}, []domain.Block{}

		for _, block := range message.Blocks {
			if block.Kind == domain.BlockToolResult {
				results = append(results, block)
			} else {
				rest = append(rest, block)
			}
		}

		ordered = append(ordered, domain.Message{Role: message.Role, Blocks: append(results, rest...)})
	}

	return ordered
}

// streamTo forwards model text to whoever follows the task; nil when nobody can.
func (r *Runner) streamTo(taskID string) func(string) {
	if r.deps.Notify == nil {
		return nil
	}

	return func(delta string) { r.deps.Notify.Delta(taskID, delta) }
}
