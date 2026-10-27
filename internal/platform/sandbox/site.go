package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// Start runs spec.Argv in the shared sandbox (or a throwaway container in host-mode tests)
// that keeps running after Start returns, and returns its id.
func (d *Docker) Start(ctx context.Context, spec Spec) (string, error) {
	mounts, err := d.cfg.mounts(spec)
	if err != nil {
		return "", err
	}

	if d.useShared() {
		return d.startShared(ctx, spec, mounts)
	}

	created, err := d.api.ContainerCreate(ctx, d.createOptions(spec, mounts))
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if _, err = d.api.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
		d.remove(ctx, created.ID)

		return "", fmt.Errorf("start container: %w", err)
	}

	return created.ID, nil
}

// Stop ends a site started with Start and cleans it up, whatever state it is in.
func (d *Docker) Stop(ctx context.Context, id string) {
	if strings.HasPrefix(id, sharedSitePrefix) {
		d.killShared(ctx, id)

		return
	}

	d.remove(ctx, id)
}

// Running reports whether the site or container exists and is still running.
func (d *Docker) Running(ctx context.Context, id string) bool {
	if strings.HasPrefix(id, sharedSitePrefix) {
		return d.runningShared(ctx, id)
	}

	res, err := d.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{})

	return err == nil && res.Container.State != nil && res.Container.State.Running
}

// LogTail returns the last lines the site or container wrote to its output.
func (d *Docker) LogTail(ctx context.Context, id string, lines string) string {
	if strings.HasPrefix(id, sharedSitePrefix) {
		return d.logTailShared(ctx, id, lines)
	}

	logs, err := d.api.ContainerLogs(ctx, id, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: lines})
	if err != nil {
		return ""
	}

	defer func() { _ = logs.Close() }()

	var out bytes.Buffer

	_, _ = stdcopy.StdCopy(&out, &out, logs)

	return out.String()
}
