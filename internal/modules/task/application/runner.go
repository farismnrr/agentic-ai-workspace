package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	defaultWallTime  = 15 * time.Minute
	defaultMaxSteps  = 40
	defaultRetries   = 2
	defaultMaxResult = 24_000
)

// Limits keep one task from running away with time or steps.
type Limits struct {
	Backoff   func(attempt int) time.Duration
	WallTime  time.Duration
	MaxSteps  int
	Retries   int
	MaxResult int
}

// DefaultLimits are the production limits.
func DefaultLimits() Limits {
	return Limits{
		Backoff:  func(attempt int) time.Duration { return time.Duration(attempt+1) * 3 * time.Second },
		WallTime: defaultWallTime, MaxSteps: defaultMaxSteps,
		Retries: defaultRetries, MaxResult: defaultMaxResult,
	}
}

// Deps are the runner's collaborators.
type Deps struct {
	Tasks      TaskStore
	Events     EventStore
	Transcript TranscriptStore
	Usage      UsageStore
	Models     Models
	Windows    ContextWindows
	Persona    PersonaSource
	Projects   Projects
	Tools      ToolSet
	Gate       ToolGate
	IDs        IDs
	Notify     Notifier
	// Activity, when set, is told about the steps that change or run something; optional.
	Activity ActivitySink
	// Computer says what the agent's run_command can use; optional.
	Computer Computer
	Now      func() time.Time
}

// Runner is the agent loop: it talks to the model, runs tools, and records every
// step, until the task needs the person, is finished, or fails.
type Runner struct {
	deps Deps
	lim  Limits
}

// NewRunner builds the agent loop.
func NewRunner(deps Deps, limits Limits) *Runner {
	return &Runner{deps: deps, lim: limits}
}

type stop int

const (
	stopWaiting stop = iota + 1
	stopFinished
	stopFailed
)

type outcome struct {
	text string
	hint string
	stop stop
}

func failed(text, hint string) outcome {
	return outcome{stop: stopFailed, text: text, hint: hint}
}

// Run works on a task until it must wait, finishes or fails. Every result is
// recorded in the task's thread and status; Run itself returns nothing.
func (r *Runner) Run(ctx context.Context, taskID string) {
	task, err := r.deps.Tasks.Find(ctx, "", taskID)
	if err != nil {
		return
	}

	// The task must still be recorded as stopped even if ctx was cancelled.
	settle := context.WithoutCancel(ctx)

	runCtx, cancel := context.WithTimeout(ctx, r.lim.WallTime)
	defer cancel()

	result := r.loop(runCtx, task)
	r.settle(settle, task, result)
}

func (r *Runner) loop(ctx context.Context, task domain.Task) outcome {
	status := domain.StatusWorking
	if len(task.Plan) == 0 {
		status = domain.StatusPlanning
	}

	if err := r.deps.Tasks.SetStatus(ctx, task.ID, status, r.deps.Now()); err != nil {
		return failed("I could not start this task.", "")
	}

	// A step is one model turn; working through its tool calls does not count.
	state := &runState{}

	for turns := 0; turns < r.lim.MaxSteps; {
		result, done, modelTurn := r.advance(ctx, task, state)
		if done {
			return result
		}

		if modelTurn {
			turns++
		}
	}

	return failed(fmt.Sprintf("I stopped after %d steps without finishing.", r.lim.MaxSteps),
		"Press Continue if you want me to keep going.")
}

// advance does one unit of work: answer pending tool calls, or ask the model for the next move.
func (r *Runner) advance(
	ctx context.Context, task domain.Task, state *runState,
) (result outcome, done, modelTurn bool) {
	if stopped, over := r.limitReached(ctx); over {
		return stopped, true, false
	}

	transcript, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err != nil {
		return failed("I lost track of this task's history.", "Try Continue in a moment."), true, false
	}

	if pending := pendingCalls(transcript); len(pending) > 0 {
		result, done = r.process(ctx, task, pending)

		return result, done, false
	}

	result, done = r.turn(ctx, task, state, transcript)

	return result, done, true
}

func (r *Runner) limitReached(ctx context.Context) (outcome, bool) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return failed("This task took longer than the time limit, so I paused.", "Press Continue to keep going."), true
	case ctx.Err() != nil:
		return failed("Stopped.", "Press Continue to pick up where I left off."), true
	}

	return outcome{}, false
}

// settle records how a run ended.
func (r *Runner) settle(ctx context.Context, task domain.Task, result outcome) {
	if result.stop != stopFailed {
		return
	}

	_, _ = r.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventError, Text: result.text, Hint: result.hint, Time: r.deps.Now(),
	})
	_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusFailed, r.deps.Now())
}
