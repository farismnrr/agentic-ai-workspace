package sandbox

import (
	"bytes"
	"context"
	"fmt"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// Start runs spec.Argv in a new container that keeps running after Start returns, and
// returns its id. It has exactly the same isolation and limits as Run. Stop ends it; if the
// server restarts it is removed with the other containers this server owns.
func (d *Docker) Start(ctx context.Context, spec Spec) (string, error) {
	mounts, err := d.cfg.mounts(spec)
	if err != nil {
		return "", err
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

// Stop ends a container started with Start and removes it, whatever state it is in.
func (d *Docker) Stop(ctx context.Context, id string) {
	d.remove(ctx, id)
}

// Running reports whether the container exists and is still running.
func (d *Docker) Running(ctx context.Context, id string) bool {
	res, err := d.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{})

	return err == nil && res.Container.State != nil && res.Container.State.Running
}

// LogTail returns the last lines the container wrote to its output.
func (d *Docker) LogTail(ctx context.Context, id string, lines string) string {
	logs, err := d.api.ContainerLogs(ctx, id, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: lines})
	if err != nil {
		return ""
	}

	defer func() { _ = logs.Close() }()

	var out bytes.Buffer

	_, _ = stdcopy.StdCopy(&out, &out, logs)

	return out.String()
}
