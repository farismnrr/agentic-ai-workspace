package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// idleJobs is how many jobs in a row the orchestrator has handed out since anything last got anywhere: a file
// changed, a check ran or a document was written. Reading and reporting alone is not progress.
func idleJobs(events []domain.Event) int {
	idle := 0

	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]

		switch {
		case event.Kind == domain.EventPRD, event.Kind == domain.EventAction && (len(event.Files) > 0 || event.Run != nil):
			return idle
		case event.Kind == domain.EventHandoff && event.Action == handoffAssigned:
			idle++
		}
	}

	return idle
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

func stuckText(jobs int) string {
	return fmt.Sprintf("The last %d jobs changed nothing, ran no check and wrote no document, so the team is stuck. "+
		"Stop handing out the same kind of job: change the approach, or finish and tell the person plainly what is "+
		"blocking and what is left.", jobs)
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
