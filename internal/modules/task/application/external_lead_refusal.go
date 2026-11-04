package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Service) externalOwned(ctx context.Context, userID, taskID string) (domain.Task, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return task, err
	}
	if !strings.HasPrefix(task.Source, externalLeadSource) || task.ArchivedAt != nil {
		return domain.Task{}, domain.ErrWrongState
	}
	return task, nil
}

func (s *Service) externalPlanRefusal(
	ctx context.Context, userID string, task domain.Task, r *Runner, roster *Roster, agentKey string,
) error {
	events, err := s.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return err
	}
	role, ok := roster.member(agentKey)
	if !ok {
		return refusal("ROLE_NOT_ALLOWED", "That team role is not enabled. Choose an enabled role from list_agents.")
	}
	if role.ChecksSecurity && task.ReviewMode != domain.ReviewModeDeepReview {
		return refusal("ROLE_NOT_ALLOWED", "Security review requires the owner to enable @deep-review for this task.")
	}
	if why := awaitingApproval(task, roster.Lead.Name, role, events); why != "" {
		approval, approvalErr := s.ExternalApproval(ctx, userID, task.ID)
		if approvalErr != nil {
			return approvalErr
		}
		if approval.Status == "revision-requested" {
			return refusal("APPROVAL_REQUIRED", "The person asked for changes. Update the PRD and plan, then submit them for approval again.")
		}
		return refusal("APPROVAL_REQUIRED", "The PRD is waiting for the person's approval. Ask them to approve it before assigning a building role.")
	}
	if task.Status != domain.StatusQueued && task.Status != domain.StatusPlanning && task.Status != domain.StatusWorking {
		return refusal("TASK_NOT_ACTIVE", "This task is waiting for a person or review. Resolve that first, then assign work.")
	}
	request, plan := lastSeq(events, domain.EventRequest), lastSeq(events, domain.EventPlan)
	if r.lim.RequirePlan && request >= plan {
		return refusal("PLAN_REQUIRED", "Add this request to the task plan with update_task_plan, then delegate again.")
	}
	if why := researchFirst(roster, role, events); why != "" {
		return refusal("RESEARCH_REQUIRED", why)
	}
	if idleJobs(events) >= r.lim.idleJobs() {
		return refusal("TEAM_STUCK", stuckText(r.lim.idleJobs()))
	}
	return nil
}

func refusal(code, remedy string) error { return fmt.Errorf("%s: %s", code, remedy) }
