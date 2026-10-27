package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

// supersedePlanMutation reopens a submitted plan when its source material changes.
func supersedePlanMutation(
	work domain.ExternalWork, patch *WorkPatch, task *TaskPatch, events []domain.Event,
) (*WorkPatch, *TaskPatch, []domain.Event) {
	if work.PlanState != domain.PlanReadyForApproval || !work.PlanState.CanMoveTo(domain.PlanDraft) {
		return patch, task, events
	}
	if patch == nil {
		patch = &WorkPatch{}
	}
	phase := domain.PlanDraft
	patch.PlanState, patch.SupersedePlan = &phase, true
	if task == nil {
		task = &TaskPatch{}
	}
	if task.Status == nil {
		task.Status = statusPtr(domain.StatusWorking)
	}
	return patch, task, events
}
