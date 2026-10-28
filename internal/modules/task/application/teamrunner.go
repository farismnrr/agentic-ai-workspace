package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// roster loads the team for a task that asked for one. It is nil for a solo task, for a teammate, and
// when the person has nobody to hand work to, so the run then goes on as a solo one.
func (r *Runner) roster(ctx context.Context, task domain.Task) *Roster {
	if !task.Team || r.role != nil || r.deps.Team == nil || r.deps.Scopes == nil || r.deps.Delegations == nil {
		return nil
	}

	roster, err := r.deps.Team.Roster(ctx, task.UserID)
	if err != nil || len(roster.Members) == 0 {
		return nil
	}

	return &roster
}

// leading is the runner that leads a run: on a team task its steps carry the lead's name, so every step of
// the thread says who did it. It is a copy, because a Runner is shared by every task running at once.
func (r *Runner) leading(state *runState) *Runner {
	if state.roster == nil {
		return r
	}

	lead := *r
	lead.leads = true
	lead.deps.Events = labelledEvents{EventStore: r.deps.Events, agent: state.roster.Lead.Name}

	return &lead
}

// child is the runner a teammate works with: the same loop, with its own transcript, only the
// tools its role allows, and every step it records labelled with its name and job id.
func (r *Runner) child(role AgentRole, d Delegation) *Runner {
	deps := r.deps
	deps.Transcript = r.deps.Scopes.Scope(d.ID)
	deps.Events = labelledEvents{EventStore: r.deps.Events, agent: d.RoleName, job: d.ID}
	deps.Gate = roleGate{ToolGate: r.deps.Gate, allow: roleToolSet(role.Tools)}
	deps.Team, deps.Scopes, deps.Delegations = nil, nil, nil

	return &Runner{deps: deps, lim: r.lim, role: &role, job: d.ID, scope: d.Scope, since: d.StartSeq}
}

// labelledEvents puts the teammate's name and delegation id on every step they add to the thread.
type labelledEvents struct {
	EventStore
	agent string
	job   string
}

func (l labelledEvents) AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error) {
	if event.Agent == "" {
		event.Agent = l.agent
	}
	if event.Job == "" {
		event.Job = l.job
	}

	return l.EventStore.AppendEvent(ctx, taskID, event) //nolint:wrapcheck // a pass-through
}

// roleGate narrows the person's tool switches to what the role may use.
type roleGate struct {
	ToolGate
	allow map[string]bool
}

func (g roleGate) Enabled(ctx context.Context, userID string) (map[string]bool, error) {
	enabled, err := g.ToolGate.Enabled(ctx, userID)
	if err != nil {
		return nil, err //nolint:wrapcheck // a pass-through
	}

	narrowed := make(map[string]bool, len(enabled))
	for name, on := range enabled {
		narrowed[name] = on && g.allow[name]
	}

	return narrowed, nil
}

// toolRequestEgress is asking for a website. A role that may run commands may ask; the server only offers the tool
// when egress grants are on, and a person decides.
const toolRequestEgress = "request_egress"

// roleToolSet is the tools a role may use: its own list, plus asking for a website when it can run commands.
func roleToolSet(names []string) map[string]bool {
	set := toSet(names)
	if set[toolRunCommand] {
		set[toolRequestEgress] = true
	}

	return set
}

func toSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}

	return set
}

// drive runs the loop until the run waits, finishes or fails, or its steps run out. A teammate's
// loop is the same as the orchestrator's, but it also stops after its own share of steps.
func (r *Runner) drive(ctx context.Context, task domain.Task, state *runState, limitText string) outcome {
	for state.budget.has() && state.segment > 0 {
		result, done, modelTurn := r.advance(ctx, task, state)
		if done {
			return result
		}

		if modelTurn {
			state.budget.take()
			state.segment--
		}
	}

	return failed(limitText, "Press Continue if you want me to keep going.")
}

func stepsText(who string, steps int) string {
	return fmt.Sprintf("%s stopped after %d steps without finishing.", who, steps)
}
