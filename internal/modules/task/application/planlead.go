package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// leadTools are the tools of the app's loop that the lead's program gets from the server.
var leadTools = []string{toolDelegate, toolAskUser, toolUpdatePlan}

// leadSession is one turn of a team's lead on the person's own plan. The program thinks; the server keeps the
// team's rules: the same calls the app's own loop answers (delegate, ask_user, update_plan) are answered by the
// same code, so the roster, the person's approval of the PRD and the receipt work as they do for any model.
type leadSession struct {
	inline  *Runner
	lead    *Runner
	state   *runState
	results *capturedResults
	// stopped is how the run must end when the program cannot go on: the person has to answer first, or time ran out.
	stopped *outcome
	task    domain.Task
	mu      sync.Mutex
}

// leadOf starts the lead's session when the task has a team; nil when it does not and runs alone.
func (r *Runner) leadOf(ctx context.Context, task domain.Task) *leadSession {
	roster := r.roster(ctx, task)
	if roster == nil {
		return nil
	}

	steps := r.lim.steps(true)
	state := &runState{budget: &budget{left: steps}, segment: steps, roster: roster}
	lead := r.leading(state)
	results := &capturedResults{answers: map[string]capturedResult{}}

	inline := *lead
	inline.deps.Transcript = results

	return &leadSession{inline: &inline, lead: lead, state: state, results: results, task: task}
}

// shape makes the request the lead's: the team's prompt, the server's tools, and only the looking tools of its own.
func (l *leadSession) shape(ctx context.Context, req CLIRequest) CLIRequest {
	req.Prompt = l.inline.leadPrompt(ctx, l.task, *l.state.roster, req.Prompt)
	req.Bridge = l.bridge()
	req.Limited = true
	req.Tools = l.inline.lookTools(ctx, l.task.UserID)

	return req
}

// lookTools are the project tools that only look, after the person's own switches: the lead never changes anything.
func (r *Runner) lookTools(ctx context.Context, userID string) []string {
	enabled, err := r.deps.Gate.Enabled(ctx, userID)
	if err != nil {
		return nil
	}

	names := []string{}

	for name, on := range enabled {
		if on && r.deps.Tools.Risk(name) == domain.RiskNone {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}

// bridge is what the program may call on the server.
func (l *leadSession) bridge() *CLIBridge {
	tools := []CLITool{toolOfDef(delegateTool(l.state.roster.Members))}

	for _, def := range agentTools() {
		if def.Name == toolAskUser || def.Name == toolUpdatePlan {
			tools = append(tools, toolOfDef(def))
		}
	}

	return &CLIBridge{Tools: tools, Call: l.call}
}

func toolOfDef(def domain.ToolDef) CLITool {
	return CLITool{Name: def.Name, Description: def.Description, Schema: def.Schema}
}

// call answers one call of the program like the loop answers one of the model's.
func (l *leadSession) call(ctx context.Context, name string, input json.RawMessage) (string, bool) {
	if !contains(leadTools, name) {
		return "There is no tool called " + name + ".", true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stopped != nil {
		return "Stop now: the run is over for this turn.", true
	}

	id := l.callID(ctx, name, input)

	out, done := l.inline.handle(ctx, l.task, l.state, Call{ID: id, Name: name, Input: input})
	if done {
		l.stopped = &out

		return stopText(out), out.stop == stopFailed
	}

	got := l.results.take(id)

	return got.text, got.failed
}

// stopText tells the program why it has to end its turn.
func stopText(out outcome) string {
	if out.stop == stopWaiting {
		return "The person has to answer first; it is shown to them now. End your turn now without more calls. " +
			"Their answer arrives as your next message. If a teammate is waiting, call delegate again then with the " +
			"same role and brief to continue it."
	}

	return out.text + " End your turn now."
}

// callID names a call. A delegation keeps its name between turns, so asking again for the same role and brief
// goes on with a teammate that was waiting for the person; a job already finished is a new one.
func (l *leadSession) callID(ctx context.Context, name string, input json.RawMessage) string {
	sum := sha256.Sum256(append([]byte(name+"\x00"), input...))
	base := "lead-" + hex.EncodeToString(sum[:8])

	if name != toolDelegate || l.inline.deps.Delegations == nil {
		return base + "-" + strconv.FormatInt(l.inline.deps.Now().UnixNano(), 36)
	}

	for n := 0; ; n++ {
		id := base + "-" + strconv.Itoa(n)

		d, found, err := l.inline.deps.Delegations.FindDelegation(ctx, l.task.ID, id)
		if err != nil || !found || d.Status == DelegationWorking || d.Status == DelegationWaiting {
			return id
		}
	}
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}

	return false
}
