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
func (r *Runner) turn(ctx context.Context, task domain.Task, transcript []domain.Message) (outcome, bool) {
	client, err := r.deps.Models.Client(ctx, task.UserID, task.ProviderID)
	if err != nil {
		return failed("I cannot reach the AI model you chose.", "Check Settings, AI models."), true
	}

	offered, err := r.offered(ctx, task.UserID)
	if err != nil {
		return failed("I cannot check which tools you switched on.", "Try Continue in a moment."), true
	}

	reply, err := r.ask(ctx, client, Request{
		Model: task.Model, System: r.system(ctx, task), Messages: orderBlocks(transcript), OnText: r.streamTo(task.ID),
		Tools: offered,
	})
	if err != nil {
		return r.modelFailure(ctx, task, err), true
	}

	_ = r.deps.Usage.AddUsage(ctx, UsageRecord{
		At: r.deps.Now(), TaskID: task.ID, UserID: task.UserID, ProviderID: task.ProviderID, Model: task.Model,
		Usage: reply.Usage,
	})

	return r.record(ctx, task, reply)
}

func (r *Runner) record(ctx context.Context, task domain.Task, reply Reply) (outcome, bool) {
	if len(reply.Blocks) == 0 {
		return failed("The AI model sent an empty answer.", "Try Continue, or pick another model."), true
	}

	calls, texts := false, []string{}

	for _, block := range reply.Blocks {
		_ = r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleAssistant, block)

		switch block.Kind {
		case domain.BlockToolUse:
			calls = true
		case domain.BlockText:
			texts = append(texts, strings.TrimSpace(block.Text))
		case domain.BlockToolResult:
		}
	}

	say := strings.TrimSpace(strings.Join(texts, "\n\n"))
	if calls {
		r.say(ctx, task.ID, say)

		return outcome{}, false
	}

	// No tool calls: the model considers itself done.
	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if !workDone(events) {
		return r.answer(ctx, task, say), true
	}

	return r.complete(ctx, task, say, buildProof(events, nil)), true
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

func (r *Runner) modelFailure(ctx context.Context, task domain.Task, err error) outcome {
	if ctx.Err() != nil {
		result, _ := r.limitReached(ctx, task)

		return result
	}

	message := "The AI model could not answer."
	if modelErr, ok := domain.AsModelError(err); ok {
		message = "The AI model said: " + truncate(modelErr.Message, maxModelError)
	}

	return failed(message, "Check your key and model under Settings, AI models, then press Continue.")
}

func (r *Runner) system(ctx context.Context, task domain.Task) string {
	persona, _ := r.deps.Persona.Persona(ctx, task.UserID)

	project, err := r.deps.Projects.Name(ctx, task.UserID, task.ProjectID)
	if err != nil {
		project = "your project"
	}

	var installed []string
	if r.deps.Computer != nil {
		installed = r.deps.Computer.Toolchains(ctx)
	}

	return systemPrompt(project, persona, installed)
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
