package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const (
	millisPerCore = 1000
	mebibyte      = 1 << 20
)

// toolboxDir is the shared, read-only root every sandbox starts from.
func toolboxDir(cfg config.Config) string {
	dir, err := filepath.Abs(filepath.Join(cfg.SandboxDir, "general"))
	if err != nil {
		return filepath.Join(cfg.SandboxDir, "general")
	}

	return dir
}

// errLimitsOff is what a task is told when resource limits cannot be enforced.
var errLimitsOff = errors.New("the server cannot enforce resource limits for checks, so it will not run them; " +
	"its owner can fix this by running the server as a systemd service with Delegate=yes, " +
	"or accept weaker limits with SANDBOX_LIMITS=relaxed")

// newSandboxEngine builds the infrastructure all runners share. When cgroups are not available the
// default is to refuse to run checks rather than run them without real CPU and memory limits.
func newSandboxEngine(cfg config.Config) *codingrunner.Engine {
	limits := sandbox.DefaultLimits()
	if cfg.SandboxMemoryMB > 0 {
		limits.MemoryBytes = uint64(cfg.SandboxMemoryMB) * mebibyte
	}

	if cfg.SandboxCPUs > 0 {
		limits.CPUMillis = uint64(cfg.SandboxCPUs) * millisPerCore
	}

	engine := codingrunner.EngineConfig{Toolbox: toolboxDir(cfg), Limits: limits, Size: cfg.MaxSandboxes}

	cgroups, err := sandbox.OpenCgroups()

	switch {
	case err == nil:
		engine.Cgroups = cgroups
		log.Printf("sandbox: cgroup limits on (%d MB, %d CPU per check, %d at a time)",
			limits.MemoryBytes/mebibyte, limits.CPUMillis/millisPerCore, cfg.MaxSandboxes)
	case cfg.SandboxRelaxed:
		log.Printf("sandbox: WARNING: %v; running checks with weaker limits because SANDBOX_LIMITS=relaxed", err)
	default:
		engine.Refuse = fmt.Errorf("%w (%w)", errLimitsOff, err)
		log.Printf("sandbox: %v; checks are refused until this is fixed", err)
	}

	if err := sandbox.ToolboxReady(engine.Toolbox); err != nil {
		log.Printf("sandbox: %v", err)
	}

	return codingrunner.NewEngine(engine)
}
