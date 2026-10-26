package runner

import (
	"context"
	"errors"
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

// Run starts the task and waits for it, removing the container on timeout.
func (r *Runner) Run(ctx context.Context, task domain.Task) (application.RunResult, error) {
	select {
	case r.slot <- struct{}{}:
		defer func() { <-r.slot }()
	default:
		return application.RunResult{}, domain.ErrTaskBusy
	}

	release, err := r.engine.acquire(ctx)
	if err != nil {
		return application.RunResult{}, err
	}

	defer release()

	if sizeOver(r.dir, r.quota) {
		return application.RunResult{ExitCode: -1, Output: overQuota}, nil
	}

	return r.execute(ctx, task), nil
}

const (
	overQuota  = "The workspace is over its size limit. Delete some files, then try again."
	stoppedBig = "\nStopped: the workspace grew past its size limit."
	stoppedOOM = "\nStopped: it used more memory than allowed."
)

func (r *Runner) execute(ctx context.Context, task domain.Task) application.RunResult {
	ctx, cancel := context.WithTimeout(r.bind(ctx), task.Timeout())
	defer cancel()

	out := &tail{limit: domain.MaxOutputBytes}

	docker, err := r.engine.Docker(ctx)
	if err != nil {
		return finish(ctx, err, sandbox.Result{}, out, 0)
	}

	spec := sandbox.Spec{Workspace: r.dir, Argv: task.Command}
	projectValues, err := r.projectValues(ctx)
	if err != nil {
		return application.RunResult{ExitCode: -1, Output: "could not load project environment"}
	}

	git, hasGit := r.credentials(ctx)

	hosts, granted := r.egressHosts(ctx, hasGit)
	r.engine.prepare(ctx, r.dir, &spec, docker, hosts...)
	projectValues, granting := granted.Apply(&spec, projectValues)
	baseEnv := append(environment(spec), granting...)

	if hasGit && spec.Egress != "" {
		baseEnv = append(baseEnv, gitEnvironment(git)...)
	}
	spec.Env = MergeProjectEnvironment(baseEnv, projectValues)

	over := watchSize(ctx, r.dir, r.quota, cancel)

	started := time.Now()
	res, err := docker.Run(ctx, spec, out)
	result := finish(ctx, err, res, out, time.Since(started))

	switch {
	case over.Load():
		result.TimedOut, result.ExitCode = false, -1
		result.Output += stoppedBig
	case res.OOMKilled:
		result.ExitCode = -1
		result.Output += stoppedOOM
	}

	return result
}

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
