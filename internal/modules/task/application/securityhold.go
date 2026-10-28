package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const toolStartSite = "start_site"

const (
	maxSecurityRounds  = 3
	maxArchitectRounds = 3
)

func (r *Runner) securityNotAsked(
	ctx context.Context, task domain.Task, state *runState, call Call, impact verificationImpact,
) bool {
	if !impact.security {
		return false
	}
	return r.requireReview(ctx, task, state, call, securityOf(state.roster), &state.securityAsked,
		maxSecurityRounds, hasImpactChange, "Security", impact.reason,
		"Name every affected path in Assessment and run each named critical scenario as its own exact recorded command.")
}

func (r *Runner) architectNotAsked(
	ctx context.Context, task domain.Task, state *runState, call Call, impact verificationImpact,
) bool {
	if !impact.architecture {
		return false
	}
	return r.requireReview(ctx, task, state, call, architectOf(state.roster), &state.architectAsked,
		maxArchitectRounds, hasImpactChange, "Architect", impact.reason,
		"Check the changed contracts and affected callers against the design, SOLID and Clean Architecture.")
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
		reviewer.ChecksSecurity && !r.hasCriticalEvidence(ctx, task, events, latest)
}

func (r *Runner) isReviewReport(ctx context.Context, task domain.Task, event domain.Event, reviewer AgentRole) bool {
	if event.Kind != domain.EventHandoff || event.Action != handoffReported || event.Job == "" ||
		r.deps.Delegations == nil {
		return false
	}
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, event.Job)
	return err == nil && found && d.RoleKey == reviewer.Key
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

func architectOf(roster *Roster) AgentRole {
	if roster == nil {
		return AgentRole{}
	}
	for _, member := range roster.Members {
		if member.ChecksArchitecture {
			return member
		}
	}
	return AgentRole{}
}
