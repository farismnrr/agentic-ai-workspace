package sandbox

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

const (
	sharedVolumesRoot = "/mnt/volumes"
	sharedPidsDir     = "/run/pids"
	sharedSitePrefix  = "site-"
	rootUser          = "0:0"
	maskTmpfsOpts     = "ro,size=4k"
	pidsTmpfsOpts     = "rw,nosuid,nodev,size=16m"
)

// useShared reports whether commands and sites execute inside the single standing agentic-sandbox container.
func (d *Docker) useShared() bool {
	return d.cfg.DataVolume != "" && len(d.toolchains) > 0
}

func (d *Docker) ensureSharedSandbox(ctx context.Context) {
	if len(d.toolchains) == 0 {
		return
	}

	d.sharedMu.Lock()
	defer d.sharedMu.Unlock()

	mounts := d.sharedVolumeMounts(ctx)
	if d.sharedReady(ctx, mounts) {
		return
	}

	d.remove(ctx, SharedSandboxName)

	enabled := true
	created, err := d.api.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: SharedSandboxName,
		Config: &container.Config{
			Image: d.cfg.Image, User: rootUser,
			Entrypoint: []string{"sleep", "infinity"}, NetworkDisabled: true,
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "none", ReadonlyRootfs: true, Init: &enabled, Runtime: d.cfg.Runtime,
			CapDrop: []string{"ALL"},
			CapAdd: []string{
				"SYS_ADMIN", "NET_ADMIN", "SETUID", "SETGID", "KILL",
				"DAC_OVERRIDE", "DAC_READ_SEARCH", "CHOWN", "FOWNER",
			},
			SecurityOpt: []string{"seccomp=unconfined", "apparmor=unconfined"},
			Mounts:      mounts,
			Tmpfs: map[string]string{
				"/tmp": tmpfsOpts, Cache: maskTmpfsOpts, Home: maskTmpfsOpts,
				Egress: maskTmpfsOpts, SiteDir: maskTmpfsOpts, ViewDir: maskTmpfsOpts,
				BridgeDir: maskTmpfsOpts, sharedPidsDir: pidsTmpfsOpts,
			},
			Resources: container.Resources{
				Memory: signed(d.cfg.Limits.MemoryBytes), MemorySwap: signed(d.cfg.Limits.MemoryBytes + d.cfg.Limits.SwapBytes),
				NanoCPUs: signed(d.cfg.Limits.CPUMillis) * nanoPerMS,
			},
			RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyUnlessStopped},
		},
	})
	if err == nil {
		_, _ = d.api.ContainerStart(ctx, created.ID, client.ContainerStartOptions{})
	}
}

func (d *Docker) sharedVolumeMounts(ctx context.Context) []mount.Mount {
	candidates := []string{"agentic-data", "masihawam-data", d.cfg.DataVolume}
	var seen []string
	var mounts []mount.Mount

	for _, vol := range candidates {
		if vol == "" || slices.Contains(seen, vol) {
			continue
		}
		seen = append(seen, vol)

		if _, err := d.api.VolumeInspect(ctx, vol, client.VolumeInspectOptions{}); err == nil {
			mounts = append(mounts, mount.Mount{
				Type: mount.TypeVolume, Source: vol, Target: sharedVolumesRoot + "/" + vol,
			})
		}
	}

	return mounts
}

func (d *Docker) sharedReady(ctx context.Context, want []mount.Mount) bool {
	res, err := d.api.ContainerInspect(ctx, SharedSandboxName, client.ContainerInspectOptions{})
	if err != nil || !validSharedContainer(res.Container) {
		return false
	}

	if img, err := d.api.ImageInspect(ctx, d.cfg.Image); err == nil && img.ID != "" && res.Container.Image != img.ID {
		return false
	}

	return hasMounts(res.Container.Mounts, want)
}

func validSharedContainer(c container.InspectResponse) bool {
	if c.State == nil || !c.State.Running || c.Config == nil || c.HostConfig == nil {
		return false
	}

	return c.Config.User == rootUser && hasCap(c.HostConfig.CapAdd, "CHOWN")
}

func hasMounts(mounts []container.MountPoint, want []mount.Mount) bool {
	have := make([]string, 0, len(mounts))
	for _, m := range mounts {
		have = append(have, m.Destination)
	}

	for _, m := range want {
		if !slices.Contains(have, m.Target) {
			return false
		}
	}

	return true
}

func hasCap(caps []string, name string) bool {
	return slices.Contains(caps, name) || slices.Contains(caps, "CAP_"+name)
}

func (d *Docker) runShared(ctx context.Context, spec Spec, mounts []mount.Mount, out io.Writer) (Result, error) {
	d.ensureSharedSandbox(ctx)

	runID := fmt.Sprintf("run-%s-%d", strings.ToLower(d.cfg.Owner), containerSeq.Add(1))
	defer d.killShared(ctx, runID)

	created, err := d.api.ExecCreate(ctx, SharedSandboxName, client.ExecCreateOptions{
		User:         rootUser,
		AttachStdin:  spec.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Env:          containerEnv(spec),
		WorkingDir:   Workspace,
		Cmd:          d.sharedCmd(runID, spec, mounts, ""),
	})
	if err != nil {
		return Result{}, fmt.Errorf("create exec: %w", err)
	}

	attached, err := d.api.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return Result{}, fmt.Errorf("attach exec: %w", err)
	}

	copied := make(chan struct{})
	go func() {
		defer close(copied)
		_, _ = stdcopy.StdCopy(out, out, attached.Reader)
	}()

	if spec.Stdin != nil {
		go feed(attached.HijackedResponse, spec.Stdin)
	}

	var runErr error
	select {
	case <-copied:
	case <-ctx.Done():
		runErr = fmt.Errorf("stopped: %w", ctx.Err())
		d.killShared(ctx, runID)
	}

	select {
	case <-copied:
	case <-time.After(drainWait):
	}

	attached.Close()
	<-copied

	if runErr != nil {
		return Result{}, runErr
	}

	inspected, err := d.api.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return Result{}, fmt.Errorf("inspect exec: %w", err)
	}

	return Result{ExitCode: inspected.ExitCode}, nil
}
