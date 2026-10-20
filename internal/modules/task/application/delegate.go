package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Handoff event actions: the thread shows a card for each.
const (
	handoffAssigned = "assigned"
	handoffReported = "reported"
	handoffFailed   = "failed"
)

// delegate hands a job to a teammate and runs them to the end of it inside this run. It returns
// done when the run has to stop: the teammate waits for the person, or time or the person ran out.
func (r *Runner) delegate(ctx context.Context, task domain.Task, state *runState, call Call) (outcome, bool) {
	args, ok := parseDelegate(call.Input)
	if !ok || state.roster == nil {
		r.result(ctx, task, call.ID, "delegate needs a role and a brief, and works only on a task with a team.", true)

		return outcome{}, false
	}

	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, call.ID)
	if err != nil {
		return failed("I lost track of the team's work.", "Try Continue in a moment."), true
	}

	switch {
	case found && (d.Status == DelegationDone || d.Status == DelegationFailed):
		r.result(ctx, task, call.ID, d.Report, d.Status == DelegationFailed)

		return outcome{}, false
	case !found:
		if d, ok = r.startDelegation(ctx, task, state, call, args); !ok {
			return outcome{}, false
		}
	}

	return r.runDelegation(ctx, task, state, call, d)
}

// startDelegation records the job and seeds the teammate's conversation with the brief.
func (r *Runner) startDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, args delegateArgs,
) (Delegation, bool) {
	role, ok := state.roster.member(args.Role)
	if !ok {
		r.result(ctx, task, call.ID, "There is no teammate called "+args.Role+". "+teammateKeys(state.roster), true)

		return Delegation{}, false
	}

	events, _ := r.deps.Events.EventsAfter(ctx, task.ID, 0)
	if r.clarifyMissing(ctx, task, state, call, events) || r.planMissing(ctx, task, state, call, events) {
		return Delegation{}, false
	}

	if why := awaitingApproval(task, state.roster.Lead.Name, role, events); why != "" {
		r.result(ctx, task, call.ID, why, true)

		return Delegation{}, false
	}

	if why := researchFirst(state.roster, role, events); why != "" {
		r.result(ctx, task, call.ID, why, true)

		return Delegation{}, false
	}

	if idleJobs(events) >= r.lim.idleJobs() {
		r.result(ctx, task, call.ID, stuckText(r.lim.idleJobs()), true)

		return Delegation{}, false
	}

	providerID, model, effort := r.pickModel(ctx, task, role)
	effort = r.fitEffort(ctx, task.UserID, providerID, model, effort, args.Effort)
	d := Delegation{
		ID: call.ID, RoleKey: role.Key, RoleName: role.Name, ProviderID: providerID, Model: model, Effort: effort,
		Brief: truncate(args.Brief, maxBrief), Status: DelegationWorking,
	}

	seed := domain.Block{Kind: domain.BlockText, Text: r.briefWithPlan(ctx, task, args)}
	if err := r.deps.Scopes.Scope(d.ID).AppendBlock(ctx, task.ID, domain.RoleUser, seed); err != nil {
		r.result(ctx, task, call.ID, "I could not brief "+role.Name+".", true)

		return Delegation{}, false
	}

	if n := len(events); n > 0 {
		d.StartSeq = events[n-1].Seq
	}

	if err := r.deps.Delegations.StartDelegation(ctx, task.ID, d, r.deps.Now()); err != nil {
		r.result(ctx, task, call.ID, "I could not hand the job to "+role.Name+".", true)

		return Delegation{}, false
	}

	r.handoff(ctx, task, shownWith(d, args.Effort), handoffAssigned, "Asked the "+role.Name+" to take this on", d.Brief)
	r.trackPlan(ctx, task, d, domain.PlanActive, "")

	return d, true
}

// runDelegation lets the teammate work until they report, fail or have to wait for the person.
func (r *Runner) runDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, d Delegation,
) (outcome, bool) {
	role, ok := state.roster.member(d.RoleKey)
	if !ok {
		r.endDelegation(ctx, task, call, d, DelegationFailed, d.RoleName+" is no longer on the team.")

		return outcome{}, false
	}

	_ = r.deps.Delegations.SetDelegationStatus(ctx, task.ID, d.ID, DelegationWorking)

	teammate := task
	teammate.ProviderID, teammate.Model, teammate.Effort = d.ProviderID, d.Model, d.Effort
	sub := &runState{budget: state.budget, segment: r.lim.delegateSteps()}

	worker := r.child(role, d)
	result := r.doJob(ctx, worker, teammate, sub, role, d)
	if stopped, over := r.limitReached(ctx); over {
		return stopped, true
	}

	switch result.stop {
	case stopWaiting:
		_ = r.deps.Delegations.SetDelegationStatus(ctx, task.ID, d.ID, DelegationWaiting)

		return result, true
	case stopFinished:
		report, err := r.deliver(ctx, task, d, role, result)
		if err != nil {
			return failed("The teammate's document could not be saved.", "Press Continue to retry saving it."), true
		}
		r.endDelegation(ctx, task, call, d, DelegationDone, report+r.facts(ctx, task, d))
	case stopFailed:
		r.endDelegation(ctx, task, call, d, DelegationFailed, d.RoleName+" could not finish: "+result.text+stuckAdvice)
	}

	return outcome{}, false
}

// endDelegation closes the job, shows the person how it ended, and answers the delegate call.
func (r *Runner) endDelegation(
	ctx context.Context, task domain.Task, call Call, d Delegation, status, report string,
) {
	_ = r.deps.Delegations.EndDelegation(ctx, task.ID, d.ID, status, report, r.deps.Now())

	action, text := handoffReported, "The "+d.RoleName+" reported back"
	if status == DelegationFailed {
		action, text = handoffFailed, "The "+d.RoleName+" could not finish"
	}

	plan, detail := planStateAfter(status == DelegationFailed, report)
	r.trackPlan(ctx, task, d, plan, detail)

	r.handoff(ctx, task, d, action, text, report)
	r.result(ctx, task, call.ID, report, status == DelegationFailed)
}

func (r *Runner) handoff(ctx context.Context, task domain.Task, d Delegation, action, text, detail string) {
	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventHandoff, Text: text, Technical: detail, Action: action, Agent: d.RoleName, Model: d.Model,
		Effort: string(d.Effort), Time: r.deps.Now(),
	})
}

func teammateKeys(roster *Roster) string {
	keys := make([]string, 0, len(roster.Members))
	for _, member := range roster.Members {
		keys = append(keys, member.Key)
	}

	return "Choose one of: " + strings.Join(keys, ", ") + "."
}

func reportText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(The teammate sent no report.)"
	}

	return text
}
