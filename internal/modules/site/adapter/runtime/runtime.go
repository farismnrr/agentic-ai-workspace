// Package runtime runs sites in the sandbox. An app runs in a long-lived container with no
// network; the container publishes the app's port as a Unix socket in a folder the server
// mounted, and the server reaches the app only through that socket.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	dirMode       = 0o750
	socketName    = "s.sock"
	nameBytes     = 6
	maxSocketPath = 100
	readyTimeout  = 60 * time.Second
	readyEvery    = 500 * time.Millisecond
	probeTimeout  = 2 * time.Second
	logLines      = "40"
)

var errPathTooLong = errors.New("the data folder path is too long for a Unix socket")

// Folders finds a project's folder.
type Folders interface {
	Dir(projectID string) (string, error)
}

// Runtime implements application.Runtime on the shared sandbox engine.
type Runtime struct {
	engine  *codingrunner.Engine
	folders Folders
	base    string
}

// New keeps one socket folder per project under base (an absolute path inside the data folder).
func New(engine *codingrunner.Engine, folders Folders, base string) *Runtime {
	return &Runtime{engine: engine, folders: folders, base: base}
}

func (r *Runtime) dir(projectID string) string {
	sum := sha256.Sum256([]byte(projectID))

	return filepath.Join(r.base, hex.EncodeToString(sum[:nameBytes]))
}

// Socket is the Unix socket the app is reachable through.
func (r *Runtime) Socket(projectID string) string {
	return filepath.Join(r.dir(projectID), socketName)
}

// Start runs the command and returns once the app answers HTTP on its port.
func (r *Runtime) Start(ctx context.Context, spec application.StartSpec) (string, error) {
	docker, err := r.engine.Docker(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrNoSandboxes, err)
	}

	project, err := r.folders.Dir(spec.ProjectID)
	if err != nil {
		return "", fmt.Errorf("project folder: %w", err)
	}

	dir := r.dir(spec.ProjectID)
	if len(r.Socket(spec.ProjectID)) > maxSocketPath {
		return "", fmt.Errorf("%w: %s", errPathTooLong, dir)
	}

	if err = os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create site folder: %w", err)
	}

	_ = os.Remove(r.Socket(spec.ProjectID))

	port := strconv.Itoa(spec.Port)
	sandboxSpec := r.engine.SiteSpec(ctx, project, docker, []string{"bash", "-c", spec.Command},
		[]string{"SITE_PORT=" + port, "PORT=" + port, "HOST=127.0.0.1", "BASE_PATH=" + spec.BasePath})
	sandboxSpec.Site = dir

	id, err := docker.Start(ctx, sandboxSpec)
	if err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	if err = r.waitReady(ctx, spec.ProjectID, func() bool { return docker.Running(ctx, id) }); err != nil {
		logs := docker.LogTail(ctx, id, logLines)
		docker.Stop(ctx, id)

		return "", fmt.Errorf("%w: %s", err, logs)
	}

	return id, nil
}

// waitReady polls the socket until the app answers any HTTP response, the container
// stops, or the time is up.
func (r *Runtime) waitReady(ctx context.Context, projectID string, running func() bool) error {
	client := Client(r.Socket(projectID), probeTimeout)
	deadline := time.NewTimer(readyTimeout)
	tick := time.NewTicker(readyEvery)

	defer deadline.Stop()
	defer tick.Stop()

	for {
		if !running() {
			return fmt.Errorf("%w (it stopped by itself)", domain.ErrNotStarted)
		}

		if probe(ctx, client) {
			return nil
		}

		select {
		case <-tick.C:
		case <-deadline.C:
			return domain.ErrNotStarted
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", domain.ErrNotStarted, ctx.Err())
		}
	}
}

// Stop ends the site's container.
func (r *Runtime) Stop(ctx context.Context, containerID string) {
	if containerID == "" {
		return
	}

	if docker, err := r.engine.Docker(ctx); err == nil {
		docker.Stop(ctx, containerID)
	}
}

// Alive reports whether the container is still running.
func (r *Runtime) Alive(ctx context.Context, containerID string) bool {
	docker, err := r.engine.Docker(ctx)

	return err == nil && containerID != "" && docker.Running(ctx, containerID)
}

// Client returns an HTTP client that talks to one Unix socket whatever URL it is given.
func Client(socket string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
			DisableKeepAlives: true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// probe reports whether the app answers an HTTP request, whatever the status.
func probe(ctx context.Context, client *http.Client) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://site/", http.NoBody)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	_ = resp.Body.Close()

	return true
}
