package runner

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// poolWait is how long a task queues for a free place before the server says it is busy.
const poolWait = 30 * time.Second

// EngineConfig is what every runner shares.
type EngineConfig struct {
	// Cgroups holds each task to real CPU, memory and process limits; nil means rlimits only.
	Cgroups *sandbox.Cgroups
	// Refuse, when set, makes every task fail with this reason instead of running unprotected.
	Refuse error
	// Toolbox is the read-only folder with the programs tasks run (an absolute path).
	Toolbox string
	Limits  sandbox.Limits
	// Size is how many sandboxes may run at once across every workspace.
	Size int
}

// Engine is the sandbox infrastructure all runners share: the toolbox, the limits,
// the cgroups and the pool that bounds how many sandboxes run at once, so one
// person's checks cannot starve everyone else of CPU and memory.
type Engine struct {
	places chan struct{}
	cfg    EngineConfig
}

// NewEngine builds the shared infrastructure.
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{places: make(chan struct{}, max(cfg.Size, 1)), cfg: cfg}
}

// acquire waits for a free place and returns the function that gives it back.
func (e *Engine) acquire(ctx context.Context) (func(), error) {
	wait := time.NewTimer(poolWait)
	defer wait.Stop()

	select {
	case e.places <- struct{}{}:
		return func() { <-e.places }, nil
	case <-wait.C:
		return nil, domain.ErrServerBusy
	case <-ctx.Done():
		return nil, domain.ErrServerBusy
	}
}

// group creates the cgroup for one task, or nil when the server has none.
func (e *Engine) group() (*sandbox.Group, error) {
	if e.cfg.Cgroups == nil {
		return nil, nil //nolint:nilnil // no cgroups configured is a normal, limited mode
	}

	return e.cfg.Cgroups.New(e.cfg.Limits) //nolint:wrapcheck // the sandbox package's errors are descriptive
}
