package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	codingbackup "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	codingfs "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

const workspaceMode = 0o750

var (
	userIDPattern  = regexp.MustCompile(`^[a-f0-9]{8,64}$`)
	errBadUserID   = errors.New("invalid user id")
	errOutsideBase = errors.New("workspace path escapes the workspaces folder")
)

// workspaces gives every person a private coding workspace at
// DATA_DIR/workspaces/<user id>. Files, undo history, task definitions and the
// task sandbox are never shared: nobody can name another person's folder.
type workspaces struct {
	backups *codingbackup.Store
	engine  *codingrunner.Engine
	cache   map[string]codingmcp.Services
	base    string
	mu      sync.Mutex
}

func newWorkspaces(cfg config.Config, engine *codingrunner.Engine) (*workspaces, error) {
	base, err := filepath.Abs(cfg.WorkspacesDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workspaces folder: %w", err)
	}

	if err = os.MkdirAll(base, workspaceMode); err != nil {
		return nil, fmt.Errorf("create workspaces folder: %w", err)
	}

	backups, err := codingbackup.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open backups: %w", err)
	}

	return &workspaces{backups: backups, engine: engine, base: base, cache: map[string]codingmcp.Services{}}, nil
}

// For returns the use cases bound to one person's workspace, creating it on first use.
func (w *workspaces) For(userID string) (codingmcp.Services, error) {
	dir, err := w.dir(userID)
	if err != nil {
		return codingmcp.Services{}, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if services, ok := w.cache[userID]; ok {
		return services, nil
	}

	if err = os.MkdirAll(dir, workspaceMode); err != nil {
		return codingmcp.Services{}, fmt.Errorf("create workspace: %w", err)
	}

	files, err := codingfs.Open(dir)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("open workspace: %w", err)
	}

	services := codingmcp.Services{
		Reader: codingapp.NewService(files),
		Editor: codingapp.NewEditor(files, files, w.backups, opaque.Generator{}),
		Tasks:  codingapp.NewTasks(files, codingrunner.New(dir, w.engine)),
	}
	w.cache[userID] = services

	return services, nil
}

// Erase deletes a person's workspace; it runs when their account is deleted.
func (w *workspaces) Erase(_ context.Context, userID string) error {
	dir, err := w.dir(userID)
	if err != nil {
		return err
	}

	w.mu.Lock()
	delete(w.cache, userID)
	w.mu.Unlock()

	w.engine.Forget(dir)

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}

	return nil
}

// dir maps a user id to its folder. Only ids the server generated are accepted,
// and the result is checked to stay directly under the workspaces folder.
func (w *workspaces) dir(userID string) (string, error) {
	if !userIDPattern.MatchString(userID) {
		return "", errBadUserID
	}

	dir := filepath.Join(w.base, userID)
	if filepath.Dir(dir) != w.base {
		return "", errOutsideBase
	}

	return dir, nil
}
