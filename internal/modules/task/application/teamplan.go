package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// assignedCount is how many jobs the orchestrator has handed out on this task so far.
func assignedCount(events []domain.Event) int {
	count := 0

	for _, event := range events {
		if event.Kind == domain.EventHandoff && event.Action == handoffAssigned {
			count++
		}
	}

	return count
}

const maxDocument = 12_000

// deliver is what the orchestrator reads of a teammate's report. A document that came with it, such as the PRD,
// is saved on the task and shown to the person, so everything after works from the same one.
func (r *Runner) deliver(ctx context.Context, task domain.Task, d Delegation, result outcome) string {
	text := reportText(result.text)

	document := strings.TrimSpace(result.document)
	if document == "" {
		return text
	}

	document = truncate(document, maxDocument)
	_ = r.deps.Tasks.SavePRD(ctx, task.ID, document, r.deps.Now())
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventPRD, Text: "The " + d.RoleName + " wrote the PRD", Technical: document, Agent: d.RoleName,
		Time: r.deps.Now(),
	})

	return text + "\n\nThe PRD is saved and shown to the person:\n" + document
}

func capText(limit int) string {
	return fmt.Sprintf("The team has already been given %d jobs on this task. Do the rest yourself, or finish "+
		"and tell the person what is left.", limit)
}

// briefWithPlan is what a teammate reads first: the brief, plus the team's shared plan as it stands, so
// everyone works from the same picture. The plan is read fresh, because the orchestrator updates it as it goes.
func (r *Runner) briefWithPlan(ctx context.Context, task domain.Task, args delegateArgs) string {
	text := args.briefText()

	current, err := r.deps.Tasks.Find(ctx, "", task.ID)
	if err != nil {
		return text
	}

	if current.PRD != "" {
		text += "\n\nThe team's PRD:\n" + truncate(current.PRD, maxDocument)
	}

	if len(current.Plan) == 0 {
		return text
	}

	lines := make([]string, 0, len(current.Plan))
	for _, step := range current.Plan {
		lines = append(lines, fmt.Sprintf("- [%s] %s", step.State, step.Title))
	}

	return text + "\n\nThe team's plan so far:\n" + strings.Join(lines, "\n")
}
