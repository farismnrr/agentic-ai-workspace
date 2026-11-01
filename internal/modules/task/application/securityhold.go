package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const toolStartSite = "start_site"

const maxSecurityRounds = 3

func (r *Runner) securityNotAsked(
	ctx context.Context, task domain.Task, state *runState, call Call, impact verificationImpact,
) bool {
	if task.ReviewMode != domain.ReviewModeDeepReview || !impact.security {
		return false
	}
	return r.requireReview(ctx, task, state, call, securityOf(state.roster), &state.securityAsked,
		maxSecurityRounds, hasImpactChange, "Security + Architecture", impact.reason,
		"Use ONE Security delegation. Include Security Assessment and, when architecture is affected, Architecture Assessment; name affected paths in Assessment and run applicable critical scenarios as recorded checks.")
}

func (r *Runner) requireReview(
	ctx context.Context, task domain.Task, state *runState, call Call, reviewer AgentRole,
	asked *int, maxRounds int, touches func(domain.Event) bool, label, rationale, instruction string,
) bool {
	events, err := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		state.addUnverified(label + " review status could not be read; impact is unknown. " + rationale)
		return false
	}
	if !r.changedSinceReview(ctx, task, events, reviewer, touches) {
		return false
	}
	if reviewer.Key == "" {
		state.addUnverified(label + " review is mandatory but no matching role is enabled. " + rationale)
		return false
	}
	if r.reviewRounds(ctx, task, events, reviewer) >= maxRounds || *asked >= maxRounds {
		state.addUnverified(label + " review is still missing after the review limit. " + rationale)
		return false
	}
	*asked++
	r.result(ctx, task, call.ID, "Required verification: "+rationale+" Hand the changed files to "+reviewer.Name+
		". "+instruction+" Report the checks and any unresolved finding before trying to finish.", true)
	return true
}

func (state *holds) addUnverified(reason string) {
	if state.unverified == "" {
		state.unverified = reason
		return
	}
	state.unverified += "; " + reason
}

func (r *Runner) reviewRounds(ctx context.Context, task domain.Task, events []domain.Event, reviewer AgentRole) int {
	count := 0
	for _, event := range events {
		if r.isReviewReport(ctx, task, event, reviewer) {
			count++
		}
	}
	return count
}

func (r *Runner) changedSinceReview(
	ctx context.Context, task domain.Task, events []domain.Event, reviewer AgentRole, touches func(domain.Event) bool,
) bool {
	var lastChange, lastReview int64
	var latest domain.Event
	for _, event := range events {
		switch {
		case r.isReviewReport(ctx, task, event, reviewer):
			lastReview, latest = event.Seq, event
		case !event.Undone && touches(event):
			lastChange = event.Seq
		}
	}
	return lastChange > lastReview || latest.Seq == 0 || !reviewPassed(latest) ||
		reviewer.ChecksSecurity && !r.hasCombinedSecurityEvidence(ctx, task, events, latest)
}

func (r *Runner) hasCombinedSecurityEvidence(
	ctx context.Context, task domain.Task, events []domain.Event, report domain.Event,
) bool {
	if task.ReviewMode != domain.ReviewModeDeepReview ||
		reportSection(report.Technical, "Security Assessment") == "" {
		return false
	}
	if assessImpact(task, events).architecture &&
		reportSection(report.Technical, "Architecture Assessment") == "" {
		return false
	}
	// Critical evidence is needed for actual sensitive/uncertain code, not
	// for an architectural-only edit with no security boundary or executable risk.
	if !criticalSecurityImpact(events) {
		return true
	}
	return r.hasCriticalEvidence(ctx, task, events, report)
}

func criticalSecurityImpact(events []domain.Event) bool {
	if commandImpact(events).security {
		return true
	}
	for _, file := range changedPaths(events) {
		if pathImpact(file).security {
			return true
		}
	}
	return false
}

func (r *Runner) isReviewReport(ctx context.Context, task domain.Task, event domain.Event, reviewer AgentRole) bool {
	if event.Kind != domain.EventHandoff || event.Action != handoffReported || event.Job == "" ||
		r.deps.Delegations == nil {
		return false
	}
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, event.Job)
	return err == nil && found && canonicalMemberKey(d.RoleKey) == canonicalMemberKey(reviewer.Key)
}

func securityOf(roster *Roster) AgentRole {
	if roster == nil {
		return AgentRole{}
	}
	for _, member := range roster.Members {
		if member.ChecksSecurity {
			return member
		}
	}
	return AgentRole{}
}
