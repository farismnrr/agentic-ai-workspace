package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxClarifyHolds is how often the first job of a request is held back for the question the person may need.
const maxClarifyHolds = 1

// clarifyMissing holds the first job of a request back once, so the orchestrator decides before any work starts
// whether to ask the person. The default is to ask: the person may not be technical, and a wrong guess costs a
// whole job. A request that is simple and clear needs no question, so the orchestrator may send the same job again.
// A question already asked since the request (by the orchestrator or the Planner) satisfies it.
func (r *Runner) clarifyMissing(
	ctx context.Context, task domain.Task, state *runState, call Call, events []domain.Event,
) bool {
	if !r.lim.AskFirst || state.clarifyAsked >= maxClarifyHolds {
		return false
	}

	// Only the first job of a request is held. What was asked, or handed to a teammate, since the request is read
	// from the thread, so a run that starts again after an approval is not held a second time.
	request := lastSeq(events, domain.EventRequest)
	if request == 0 || lastSeq(events, domain.EventQuestion) > request || lastSeq(events, domain.EventHandoff) > request {
		return false
	}

	state.clarifyAsked++
	r.result(ctx, task, call.ID, "Before work starts, decide whether to ask the person. By default you ask. Ask with "+
		"ask_user, one form of one to four questions, when something they would care about is open: what exactly they "+
		"want, who it is for, how it should look or behave, what must be in it, what to avoid. They may not be "+
		"technical: use plain words, give each option a short description of what it means, put your recommendation "+
		"first and mark it (Recommended), and decide technical things yourself instead of asking. If the request is "+
		"simple and clear, no question is needed: send the same job again.", true)

	return true
}
