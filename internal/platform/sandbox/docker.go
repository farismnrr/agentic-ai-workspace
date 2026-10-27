package sandbox

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

// taskLabel marks every container this server starts; its value is the owner, so a
// server only ever cleans up after itself, never after another one on the same daemon.
const taskLabel = "agentic-ai-workspace.task"

const (
	cleanupTimeout = 30 * time.Second
	ownerTagLen    = 6
)

// Config says which image tasks run in and how they reach the person's files.
type Config struct {
	// Image is what tasks run in; it is pulled once if the daemon lacks it.
	Image string
	// Runtime optionally names a hardened OCI runtime such as runsc (gVisor).
	Runtime string
	// DataVolume, when the server itself runs in a container, is the named volume
	// mounted at DataDir; workspaces are then mounted from it by sub-path.
	DataVolume string
	// Owner names this server's containers; empty picks a fresh one, so nothing is cleaned up.
	Owner string
	// DataDir is the folder that holds every workspace.
	DataDir string
	Limits  Limits
	// UID and GID own the workspace files and run the task, so edits stay writable.
	UID, GID int
}

// Docker starts one throwaway container per command.
type Docker struct {
	api        *client.Client
	toolchains []Toolchain
	cfg        Config
}

// Open connects to the Docker daemon (DOCKER_HOST or the default socket), makes
// sure the task image is present and removes containers a crashed server left behind.
func Open(ctx context.Context, cfg Config) (*Docker, error) {
	api, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	if cfg.Owner == "" {
		cfg.Owner = strings.ToLower(rand.Text()[:ownerTagLen])
	}

	d := &Docker{api: api, cfg: cfg}

	if _, err = api.Ping(ctx, client.PingOptions{}); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	if err = d.ensureImage(ctx); err != nil {
		return nil, err
	}

	d.toolchains = d.readToolchains(ctx)

	d.removeLeftovers(ctx)

	return d, nil
}

func (d *Docker) ensureImage(ctx context.Context) error {
	if _, err := d.api.ImageInspect(ctx, d.cfg.Image); err == nil {
		return nil
	}

	pull, err := d.api.ImagePull(ctx, d.cfg.Image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", d.cfg.Image, err)
	}

	if err = pull.Wait(ctx); err != nil {
		return fmt.Errorf("pull image %s: %w", d.cfg.Image, err)
	}

	return nil
}

func (d *Docker) removeLeftovers(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	list, err := d.api.ContainerList(ctx, client.ContainerListOptions{
		All: true, Filters: make(client.Filters).Add("label", taskLabel+"="+d.cfg.Owner),
	})
	if err != nil {
		return
	}

	for _, item := range list.Items {
		d.remove(ctx, item.ID)
	}
}

// remove deletes a container whatever state it is in, even when the caller gave up.
func (d *Docker) remove(ctx context.Context, id string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	_, _ = d.api.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
}

// Toolchain is a language or build tool the task image carries.
type Toolchain struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// toolchainsLabel is the image label listing its toolchains as JSON.
const toolchainsLabel = "io.masihawam.toolchains"

// Toolchains lists what the task image offers; an image without the label offers nothing known.
func (d *Docker) Toolchains() []Toolchain { return d.toolchains }

func (d *Docker) readToolchains(ctx context.Context) []Toolchain {
	res, err := d.api.ImageInspect(ctx, d.cfg.Image)
	if err != nil || res.Config == nil {
		return nil
	}

	var list []Toolchain
	if json.Unmarshal([]byte(res.Config.Labels[toolchainsLabel]), &list) != nil {
		return nil
	}

	return list
}
