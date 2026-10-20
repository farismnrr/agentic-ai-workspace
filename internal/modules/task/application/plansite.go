package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// siteToolNames are the tools a program on a plan lacks or cannot use, so the server serves them. Starting the web
// app: it is started the same way and shows up under Sites, as for any other model. Looking things up on the web:
// the program's own WebFetch runs in its container, which reaches package registries only and so refuses every
// page, while these run on the server (SearXNG for the search, a public-only fetch for the page). The screenshot runs
// there too, because a program has no browser; the picture cannot cross the bridge, so it gets the report of the page.
var siteToolNames = []string{
	toolStartSite, "stop_site", "web_search", "fetch_page", "search_images", "download_asset", "screenshot",
}

// siteSession serves the site tools to a teammate's program. Each call goes through the same code as a call of the
// app's own loop (the Security check, the person's approval, the link in the thread).
type siteSession struct {
	inline  *Runner
	state   *runState
	results *capturedResults
	stopped *outcome
	jobID   string
	tools   []CLITool
	task    domain.Task
	mu      sync.Mutex
}

// siteBridge is the site tools for this teammate's program; nil when the role has none of them.
func (r *Runner) siteBridge(task domain.Task, role AgentRole, jobID string) *siteSession {
	tools := []CLITool{}

	for _, def := range r.deps.Tools.Definitions() {
		if contains(siteToolNames, def.Name) && contains(role.Tools, def.Name) {
			tools = append(tools, toolOfDef(def))
		}
	}

	if len(tools) == 0 {
		return nil
	}

	results := &capturedResults{answers: map[string]capturedResult{}}
	inline := *r
	inline.deps.Transcript = results

	return &siteSession{
		inline: &inline, results: results, tools: tools, task: task, jobID: jobID,
		state: &runState{budget: &budget{left: r.lim.delegateSteps()}, segment: r.lim.delegateSteps()},
	}
}

func (s *siteSession) bridge() *CLIBridge {
	if s == nil {
		return nil
	}

	return &CLIBridge{Tools: s.tools, Call: s.call}
}

// call answers one call of the program like the loop answers one of the model's. The call keeps its name between
// turns, so a step the person already approved is not asked about again when the job starts over.
func (s *siteSession) call(ctx context.Context, name string, input json.RawMessage) (string, bool) {
	if !contains(siteToolNames, name) {
		return "There is no tool called " + name + ".", true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped != nil {
		return "Stop now: the job is over for this turn.", true
	}

	sum := sha256.Sum256(append([]byte(s.jobID+"\x00"+name+"\x00"), input...))
	id := "site-" + hex.EncodeToString(sum[:8])

	out, done := s.inline.handle(ctx, s.task, s.state, Call{ID: id, Name: name, Input: input})
	if done {
		s.stopped = &out

		return stopText(out), out.stop == stopFailed
	}

	got := s.results.take(id)

	return got.text, got.failed
}

// stoppedOutcome is the outcome that ends the job early, when a call asked for it.
func (s *siteSession) stoppedOutcome() (outcome, bool) {
	if s == nil || s.stopped == nil {
		return outcome{}, false
	}

	return *s.stopped, true
}
