package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// Runner executes a task's argument vector directly (never through a shell) in
// a container whose only writable folder is the workspace, one task at a time,
// with a scrubbed environment. The shared engine bounds all runners together.
type Runner struct {
	engine    *Engine
	slot      chan struct{}
	git       GitSource
	env       ProjectEnv
	reach     Reach
	projectID string
	dir       string
	quota     int64
}

// ProjectEnv supplies values for the project whose work is running.
type ProjectEnv interface {
	Values(context.Context, string) (map[string]string, error)
}

// New runs tasks with dir (an absolute path) as the workspace, on the shared engine.
func New(dir string, engine *Engine) *Runner {
	return &Runner{dir: dir, engine: engine, slot: make(chan struct{}, 1), quota: DefaultWorkspaceBytes}
}

// WithWorkspaceLimit stops a task whose workspace grows past bytes.
func (r *Runner) WithWorkspaceLimit(bytes int64) *Runner {
	r.quota = bytes

	return r
}

// WithGit lets commands act as the workspace owner on GitHub. Tasks and commands get it; the
// web apps a workspace serves never do.
func (r *Runner) WithGit(source GitSource) *Runner {
	r.git = source

	return r
}

// WithProjectEnv loads this project's values for each sandbox run.
func (r *Runner) WithProjectEnv(projectID string, source ProjectEnv) *Runner {
	r.projectID, r.env = projectID, source

	return r
}

// RunMoLab is a one-shot sandbox invocation. It never receives Project Env
// or Git credentials, and the bearer token lives only for this invocation.
func (r *Runner) RunMoLab(ctx context.Context, task domain.Task, token, host string) (application.RunResult, error) {
	return r.Run(context.WithValue(ctx, oneShotEnvKey{}, molabRun{token: token, host: host}), task)
}

type molabRun struct{ token, host string }
type oneShotEnvKey struct{}

// Run starts the task and waits for it, removing the container on timeout.
func (r *Runner) Run(ctx context.Context, task domain.Task) (application.RunResult, error) {
	releaseSlot, err := r.acquireSlot(ctx)
	if err != nil {
		return application.RunResult{}, err
	}

	defer releaseSlot()

	release, err := r.engine.acquire(ctx)
	if err != nil {
		return application.RunResult{}, err
	}

	defer release()

	if !quotaSafeGit(ctx, task) && sizeOver(r.dir, r.quota) {
		return application.RunResult{ExitCode: -1, Output: overQuota}, nil
	}

	ctx, cancel := context.WithTimeout(r.bind(ctx), task.Timeout())
	defer cancel()

	// QA/Security commands execute with an OS-enforced read-only project
	// mount. The mount, not a bounded file scan, proves that this command
	// cannot alter workspace files. Large projects can exceed snapshot
	// limits even though the mount is still immutable.
	immutable := readOnlyCheck(ctx, task)
	var before effectSnapshot
	if !immutable {
		before = snapshotEffects(ctx, r.dir)
	}
	result := r.execute(ctx, task, cancel)
	result.Command = strings.Join(task.Command, " ")
	if immutable {
		// Only attest to effects after the container has actually executed.
		// Startup failures/timeouts remain unverified, and builds with
		// writable output mounts still use ordinary effect scanning.
		result.EffectsKnown = result.ExitCode >= 0 && !result.TimedOut
		result.EffectPaths = nil
	} else {
		after := snapshotEffects(ctx, r.dir)
		result.EffectPaths, result.EffectsKnown = changedEffects(before, after)
	}
	return result, nil

}

// readOnlyCheck means the project is mounted read-only by the sandbox.
// These contexts are server-owned, not values supplied by the assistant.
// Quota-exempt Git runs use an OS read-only workspace mount; unlike normal
// Git writes, their effects can be attested as immutable. Builds/artifacts
// still have writable output folders and must never claim immutable effects.
func readOnlyCheck(ctx context.Context, task domain.Task) bool {
	return quotaSafeGit(ctx, task) || (domain.ReadOnlyWorkspace(ctx) && !domain.GitOnly(ctx) &&
		(task.Kind == "" || task.Kind == "check"))
}

const (
	overQuota  = "The workspace is over its size limit. Delete some files, then try again."
	stoppedBig = "\nStopped: the workspace grew past its size limit."
	stoppedOOM = "\nStopped: it used more memory than allowed."
)

func (r *Runner) credentials(ctx context.Context) (Git, bool) {
	if r.git == nil {
		return Git{}, false
	}

	return r.git(ctx)
}

func finish(ctx context.Context, err error, res sandbox.Result, out *tail, took time.Duration) application.RunResult {
	text, truncated := out.result()
	result := application.RunResult{Output: text, Truncated: truncated, Duration: took, ExitCode: res.ExitCode}

	switch {
	case err == nil:
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.TimedOut, result.ExitCode = true, -1
	default:
		result.ExitCode, result.Output = -1, "could not run the command: "+err.Error()
	}

	return result
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
