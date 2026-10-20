package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// recordPlan shows in the thread that the plan was set or changed, so the person can follow how the list moved.
func (r *Runner) recordPlan(ctx context.Context, task domain.Task, steps []domain.PlanStep) {
	done := 0

	var list strings.Builder

	for _, step := range steps {
		if step.State == domain.PlanDone {
			done++
		}

		fmt.Fprintf(&list, "[%s] %s\n", step.State, step.Title)
	}

	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventPlan, Text: fmt.Sprintf("Plan updated: %d of %d steps done", done, len(steps)),
		Technical: strings.TrimSpace(list.String()), Time: r.deps.Now(),
	})
}

// planPrompt shows the orchestrator the plan as it stands, so it edits that list instead of guessing at it.
func (r *Runner) planPrompt(ctx context.Context, task domain.Task) string {
	current, err := r.deps.Tasks.Find(ctx, "", task.ID)
	if err != nil || len(current.Plan) == 0 {
		return "\nThe plan (the team's todo list) is empty. Put this request in it with update_plan before you hand out " +
			"the first job.\n"
	}

	var out strings.Builder

	out.WriteString("\nThe plan now (the team's todo list; send the whole list again when you change it):\n")

	for _, step := range current.Plan {
		fmt.Fprintf(&out, "- [%s] %s", step.State, step.Title)

		if step.Detail != "" {
			out.WriteString(" (" + step.Detail + ")")
		}

		out.WriteString("\n")
	}

	return out.String()
}

// maxPlanHolds is how often the first job of a request is held back for a missing plan entry.
const maxPlanHolds = 2

// planMissing holds a job back while the person's latest request is not in the plan. The todo list is what the
// person follows, so every request is recorded in it before any teammate starts, however small the job.
func (r *Runner) planMissing(
	ctx context.Context, task domain.Task, state *runState, call Call, events []domain.Event,
) bool {
	if !r.lim.RequirePlan || state.planAsked >= maxPlanHolds {
		return false
	}

	request, plan := lastSeq(events, domain.EventRequest), lastSeq(events, domain.EventPlan)
	if request < plan || (request == 0 && plan == 0) {
		return false
	}

	state.planAsked++
	r.result(ctx, task, call.ID, "Before you hand out this job, put the person's request in the plan with update_plan: "+
		"keep the finished steps, add the steps for this request (one to three for a small job), and begin each step "+
		"with the teammate who does it. Then delegate again.", true)

	return true
}

// lastSeq is the sequence number of the newest event of a kind, or 0.
func lastSeq(events []domain.Event, kind domain.EventKind) int64 {
	var seq int64

	for _, event := range events {
		if event.Kind == kind {
			seq = event.Seq
		}
	}

	return seq
}

// blocked matches a report that starts by saying the teammate could not go on or needs something from another role.
var blocked = regexp.MustCompile(`(?is)^[\s*#_>-]*((stuck|needs)[^:\n]{0,40}):`)

// blockedReason returns what a teammate said it was stuck on or needs, or "" for a report that finished the job.
func blockedReason(report string) string {
	if !blocked.MatchString(report) {
		return ""
	}

	line, _, _ := strings.Cut(strings.TrimSpace(report), "\n")
	line = strings.Trim(line, "*#_> ")

	return truncate(line, maxStepDetail)
}

const maxStepDetail = 160

// planStateAfter says how a plan step ends: done, or failed when the job failed or the teammate reported being
// stuck or needing another role, with that line as the reason.
func planStateAfter(jobFailed bool, report string) (domain.PlanState, string) {
	if jobFailed {
		return domain.PlanFailed, ""
	}

	if reason := blockedReason(report); reason != "" {
		return domain.PlanFailed, reason
	}

	return domain.PlanDone, ""
}
