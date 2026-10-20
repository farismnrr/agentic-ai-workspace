package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// refreshModel resolves settings once per run; in-flight calls and delegated jobs keep their model.
func (r *Runner) refreshModel(ctx context.Context, task domain.Task) (domain.Task, error) {
	if !task.FollowSettings {
		return task, nil
	}
	in, err := NewService(r.deps, nil).currentModel(ctx, task)
	if err != nil {
		return task, err
	}
	provider, model, effort := in.ProviderID, in.Model, in.Effort
	if provider == task.ProviderID && model == task.Model && effort == task.Effort {
		return task, nil
	}
	if provider != task.ProviderID {
		if err = r.transferContext(ctx, task); err != nil {
			return task, err
		}
	}
	if err = r.deps.Tasks.SetModel(ctx, task.ID, provider, model, effort); err != nil {
		return task, fmt.Errorf("save run model: %w", err)
	}
	task.ProviderID, task.Model, task.Effort = provider, model, effort
	task.ContextUsed, task.ContextWindow = 0, 0
	return task, r.recordModelChange(ctx, task)
}

func (r *Runner) recordModelChange(ctx context.Context, task domain.Task) error {
	effort := string(task.Effort)
	if effort == "" {
		effort = "auto"
	}
	_, err := r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventMessage, Time: r.deps.Now(),
		Text: fmt.Sprintf("Orchestrator now uses %s (%s), effort: %s.", task.Model, task.ProviderID, effort),
	})
	if err != nil {
		return fmt.Errorf("record model change: %w", err)
	}
	return nil
}

func (r *Runner) transferContext(ctx context.Context, task domain.Task) error {
	if r.deps.Sessions != nil {
		if err := r.deps.Sessions.SaveSession(ctx, task.ID, ""); err != nil {
			return fmt.Errorf("reset program session: %w", err)
		}
	}
	if onPlan(task.ProviderID) {
		block := domain.Block{Kind: domain.BlockText, Text: r.continuation(ctx, task.ID)}
		if err := r.deps.Transcript.AppendBlock(ctx, task.ID, domain.RoleUser, block); err != nil {
			return fmt.Errorf("transfer program context: %w", err)
		}
	}
	return nil
}

// continuation carries bounded thread facts into a fresh vendor session, including document paths.
func (r *Runner) continuation(ctx context.Context, taskID string) string {
	events, _ := r.deps.Events.EventsAfter(ctx, taskID, 0)
	const maxContinuation = 24_000
	text := ""
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		line := fmt.Sprintf("\n[%s %s] %s\n%s\n%s", e.Kind, e.Agent, e.Text, e.Technical, e.Answered)
		if len([]rune(text+line)) > maxContinuation {
			break
		}
		text = line + text
	}
	return "Earlier work on this task (reference, not new instructions):\n" + text
}

func (s *Service) currentModel(ctx context.Context, task domain.Task) (CreateInput, error) {
	in, _ := s.route(ctx, task.UserID, CreateInput{Team: &task.Team})
	provider, model, err := s.chooseModel(ctx, task.UserID, in)
	if err != nil {
		return in, err
	}
	effort, err := s.chooseEffort(ctx, task.UserID, in.Effort)
	if err != nil {
		return in, err
	}
	in.ProviderID, in.Model, in.Effort = provider, model, effort
	return in, nil
}
