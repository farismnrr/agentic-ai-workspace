package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const waitDelay = 2 * time.Second

// Runner executes a task's argument vector directly (never through a shell) in
// the workspace, one task at a time, with a scrubbed environment.
type Runner struct {
	slot chan struct{}
	dir  string
}

// New runs tasks with dir as the working directory.
func New(dir string) *Runner {
	return &Runner{dir: dir, slot: make(chan struct{}, 1)}
}

// Run starts the task and waits for it, killing the whole process group on timeout.
func (r *Runner) Run(ctx context.Context, task domain.Task) (application.RunResult, error) {
	select {
	case r.slot <- struct{}{}:
		defer func() { <-r.slot }()
	default:
		return application.RunResult{}, domain.ErrTaskBusy
	}

	ctx, cancel := context.WithTimeout(ctx, task.Timeout())
	defer cancel()

	out := &tail{limit: domain.MaxOutputBytes}
	//nolint:gosec // the command is an approved, named task from the workspace owner's tasks.json
	cmd := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr, cmd.WaitDelay = r.dir, environment(), out, out, waitDelay
	isolate(cmd)

	started := time.Now()
	err := cmd.Run()

	return finish(ctx, err, out, time.Since(started)), nil
}

func finish(ctx context.Context, err error, out *tail, took time.Duration) application.RunResult {
	text, truncated := out.result()
	result := application.RunResult{Output: text, Truncated: truncated, Duration: took}

	var exit *exec.ExitError

	switch {
	case err == nil:
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.TimedOut, result.ExitCode = true, -1
	case errors.As(err, &exit):
		result.ExitCode = exit.ExitCode()
	default:
		result.ExitCode, result.Output = -1, "could not start the command: "+err.Error()
	}

	return result
}

// environment passes only what build tools need; server settings never leak to tasks.
func environment() []string {
	env := []string{"LANG=C.UTF-8", "CI=1"}
	for _, key := range []string{"PATH", "HOME", "TMPDIR"} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}

	return env
}

// tail keeps the last limit bytes written to it.
type tail struct {
	data      []byte
	limit     int
	mu        sync.Mutex
	truncated bool
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.data = append(t.data, p...)
	if len(t.data) > t.limit {
		t.data = t.data[len(t.data)-t.limit:]
		t.truncated = true
	}

	return len(p), nil
}

func (t *tail) result() (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return string(t.data), t.truncated
}
