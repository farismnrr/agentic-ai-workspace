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

func (s *Service) externalOwnedByClient(
	ctx context.Context, userID, clientID, taskID string,
) (domain.Task, error) {
	if strings.TrimSpace(clientID) == "" {
		return domain.Task{}, domain.ErrWrongState
	}
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return task, err
	}
	if task.Source != externalLeadSource+clientID {
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
	approvalWait := task.Status == domain.StatusNeedsInput &&
		externalResearchApprovalWait(task, roster, role, events)
	if why := awaitingApproval(task, roster.Lead.Name, role, events); why != "" && !approvalWait {
		approval, approvalErr := s.ExternalApproval(ctx, userID, task.ID)
		if approvalErr != nil {
			return approvalErr
		}
		if approval.Status == "revision-requested" {
			return refusal("APPROVAL_REQUIRED", "The person asked for changes. Update the PRD and plan, then submit them for approval again.")
		}
		return refusal("APPROVAL_REQUIRED", "The PRD is waiting for the person's approval. Ask them to approve it before assigning a building role.")
	}
	if !task.Status.Running() && !approvalWait {
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

func externalResearchApprovalWait(
	task domain.Task, roster *Roster, role AgentRole, events []domain.Event,
) bool {
	if !task.PlanFirst || !isApprovalResearcher(role) || roster.Lead.Name == "" ||
		!strings.HasPrefix(task.Source, externalLeadSource) {
		return false
	}
	prd := lastSeq(events, domain.EventPRD)
	if prd == 0 {
		return false
	}
	prd = max(prd, lastPlanRevisionSeq(events))
	if approvedAfter(events, task.Source, prd) {
		return false
	}
	var approval *domain.Event
	for i := range events {
		event := &events[i]
		if event.Seq > prd && event.Kind == domain.EventQuestion && event.Agent == task.Source {
			approval = event
		}
	}
	if approval == nil || !isExternalPlanApprovalQuestion(*approval) {
		return false
	}
	unanswered := make([]domain.Event, 0, 1)
	for _, event := range events {
		if event.Kind == domain.EventQuestion && event.Answered == "" {
			unanswered = append(unanswered, event)
		}
	}
	if approval.Answered == "" {
		return len(unanswered) == 1 && unanswered[0].ID == approval.ID
	}
	return len(unanswered) == 0 &&
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(approval.Answered)), "change something")
}

func isExternalPlanApprovalQuestion(event domain.Event) bool {
	if len(event.Questions) != 1 {
		return false
	}
	question := event.Questions[0]
	return event.Text == "The external orchestrator submitted the PRD. Start building?" &&
		question.Text == "Approve the PRD and start implementation?" && question.Type == domain.QuestionSingle &&
		len(question.Options) == 2 && question.Options[0].Label == startBuilding &&
		question.Options[1].Label == "Change something"
}

func isApprovalResearcher(role AgentRole) bool {
	return role.Key == "researcher" && !roleCanWrite(role) && !buildsThings(role) &&
		!role.ChecksSecurity
}

func (s *Service) externalApprovalResearcherRunner(
	ctx context.Context, task domain.Task, r *Runner, roster *Roster, agentKey string,
) (*Runner, error) {
	if task.Status != domain.StatusNeedsInput {
		return r, nil
	}
	events, err := s.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return nil, err
	}
	role, ok := roster.member(agentKey)
	if !ok || !externalResearchApprovalWait(task, roster, role, events) {
		return r, nil
	}
	scoped := *r
	scoped.approvalResearcher = true
	return &scoped, nil
}

func refusal(code, remedy string) error { return fmt.Errorf("%s: %s", code, remedy) }
