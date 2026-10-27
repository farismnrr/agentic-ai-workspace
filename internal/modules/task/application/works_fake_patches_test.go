package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

func applyFakeWorkPatch(work *domain.ExternalWork, patch *WorkPatch) {
	if patch == nil {
		return
	}
	if patch.State != nil {
		work.State = *patch.State
	}
	if patch.PlanState != nil {
		work.PlanState = *patch.PlanState
	}
	if patch.RoleKey != nil {
		work.RoleKey = *patch.RoleKey
	}
	if patch.RoleName != nil {
		work.RoleName = *patch.RoleName
	}
	if patch.EndedAt != nil {
		work.EndedAt = patch.EndedAt.Unix()
	}
	if patch.PlanRev != nil {
		work.PlanRev = *patch.PlanRev
	}
	if patch.NextStep != nil {
		work.NextStep = *patch.NextStep
	}
}

func applyFakeTaskPatch(task *domain.Task, patch *TaskPatch) {
	if patch == nil {
		return
	}
	if patch.Status != nil {
		task.Status = *patch.Status
	}
	if patch.Plan != nil {
		task.Plan = *patch.Plan
	}
	if patch.PRD != nil {
		task.PRD = *patch.PRD
	}
	if patch.Research != nil {
		task.Research = *patch.Research
	}
	if patch.Summary != nil {
		task.Summary = *patch.Summary
	}
	if patch.Proof != nil {
		task.Proof = patch.Proof
	}
}
