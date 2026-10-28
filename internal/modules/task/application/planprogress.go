package application

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// trackPlan moves the plan along with the work: the step that names a teammate becomes active when the job is
// handed to them, and done or failed when they report. The orchestrator is told to keep the plan current but
// does not reliably, and a plan that says "Not started" for finished work is worse than none. A step that names
// no teammate is left to the orchestrator.
func (r *Runner) trackPlan(ctx context.Context, task domain.Task, d Delegation, to domain.PlanState, detail string) {
	current, err := r.deps.Tasks.Find(ctx, "", task.ID)
	if err != nil || len(current.Plan) == 0 {
		return
	}

	steps := slices.Clone(current.Plan)
	from := []domain.PlanState{domain.PlanPending, domain.PlanActive}

	if to != domain.PlanActive {
		from = []domain.PlanState{domain.PlanActive, domain.PlanPending}
	}

	for _, state := range from {
		if at := stepFor(steps, d, state); at >= 0 {
			steps[at].State = to
			if detail != "" {
				steps[at].Detail = detail
			}

			r.saveTrackedPlan(ctx, task, steps)

			return
		}
	}
}

// stepFor is the first step in the given state that names the teammate (preferring one that matches its scope), or -1.
func stepFor(steps []domain.PlanStep, d Delegation, state domain.PlanState) int {
	if len(d.Scope) > 0 {
		if at := slices.IndexFunc(steps, func(step domain.PlanStep) bool {
			return stepMatchesRole(step, d, state) && stepMatchesScope(step, d.Scope)
		}); at >= 0 {
			return at
		}
	}

	return slices.IndexFunc(steps, func(step domain.PlanStep) bool {
		return stepMatchesRole(step, d, state)
	})
}

func stepMatchesRole(step domain.PlanStep, d Delegation, state domain.PlanState) bool {
	text := strings.ToLower(step.Title + " " + step.Detail)

	return step.State == state &&
		(strings.Contains(text, strings.ToLower(d.RoleName)) || strings.Contains(text, strings.ToLower(d.RoleKey)))
}

func stepMatchesScope(step domain.PlanStep, scope []string) bool {
	text := strings.ToLower(step.Title + " " + step.Detail)
	for _, prefix := range scope {
		base := strings.ToLower(path.Base(prefix))
		if strings.Contains(text, strings.ToLower(prefix)) || (len(base) >= 3 && strings.Contains(text, base)) {
			return true
		}
	}

	return false
}

// unfinishedPlan lists the steps that are neither done nor failed.
func unfinishedPlan(plan []domain.PlanStep) []string {
	var titles []string

	for _, step := range plan {
		if step.State == domain.PlanPending || step.State == domain.PlanActive {
			titles = append(titles, step.Title)
		}
	}

	return titles
}

// planStale refuses the first finish while the plan still shows unfinished steps, and tells the orchestrator to
// bring it up to date. It is for the orchestrator of a team, and asks only once, so a plan that is wrong on
// purpose cannot block the end of the run.
func (r *Runner) planStale(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	if state.planWarned || state.roster == nil {
		return false
	}

	current, err := r.deps.Tasks.Find(ctx, "", task.ID)
	if err != nil {
		return false
	}

	open := unfinishedPlan(current.Plan)
	if len(open) == 0 {
		return false
	}

	state.planWarned = true
	r.result(ctx, task, call.ID, "The plan the person sees still shows these steps as not finished: "+
		strings.Join(open, "; ")+". Call update_plan so every step shows what really happened (done, or "+
		"failed), and take out steps you did not do; then finish.", true)

	return true
}

func (r *Runner) saveTrackedPlan(ctx context.Context, task domain.Task, steps []domain.PlanStep) {
	if err := r.deps.Tasks.SavePlan(ctx, task.ID, steps, r.deps.Now()); err != nil {
		r.say(ctx, task.ID, "Plan progress could not be saved. Please retry.")
		return
	}
	if err := r.saveNotes(ctx, task); err != nil {
		r.say(ctx, task.ID, "The plan file could not be updated; reopening Plan retries the export.")
	}
}
