package sandbox

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

// containerSeq numbers containers started by this process so concurrent runs never share a name.
var containerSeq atomic.Uint64

const (
	drainWait = 2 * time.Second
	tmpfsOpts = "rw,exec,nosuid,nodev,size=512m"
	nanoPerMS = 1_000_000
)

// Run starts spec.Argv in a new container, copies its output to out and waits for
// it to end. When ctx ends first the container is killed. The container is always removed.
func (d *Docker) Run(ctx context.Context, spec Spec, out io.Writer) (Result, error) {
	mounts, err := d.cfg.Mounts(spec)
	if err != nil {
		return Result{}, err
	}

	if d.useShared() {
		return d.runShared(ctx, spec, mounts, out)
	}

	created, err := d.api.ContainerCreate(ctx, d.createOptions(spec, mounts))
	if err != nil {
		return Result{}, fmt.Errorf("create container: %w", err)
	}

	defer d.remove(ctx, created.ID)

	return d.execute(ctx, created.ID, spec.Stdin, out)
}

func (d *Docker) execute(ctx context.Context, id string, stdin io.Reader, out io.Writer) (Result, error) {
	output := client.ContainerAttachOptions{Stream: true, Stdin: stdin != nil, Stdout: true, Stderr: true}

	attached, err := d.api.ContainerAttach(ctx, id, output)
	if err != nil {
		return Result{}, fmt.Errorf("attach container: %w", err)
	}

	waiting := d.api.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNextExit})
	copied := make(chan struct{})

	go func() {
		defer close(copied)

		_, _ = stdcopy.StdCopy(out, out, attached.Reader)
	}()

	if stdin != nil {
		go feed(attached.HijackedResponse, stdin)
	}

	code, err := d.start(ctx, id, waiting)

	select {
	case <-copied:
	case <-time.After(drainWait):
	}

	attached.Close()
	<-copied

	if err != nil {
		return Result{}, err
	}

	return Result{ExitCode: code, OOMKilled: d.oomKilled(ctx, id)}, nil
}

// feed copies standard input into the container, then tells it nothing more is coming.
func feed(attached client.HijackedResponse, stdin io.Reader) {
	_, _ = io.Copy(attached.Conn, stdin)
	_ = attached.CloseWrite()
}

// start runs the container and returns its exit code; the caller removes it, which also kills it.
func (d *Docker) start(ctx context.Context, id string, waiting client.ContainerWaitResult) (int, error) {
	if _, err := d.api.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return 0, fmt.Errorf("start container: %w", err)
	}

	select {
	case res := <-waiting.Result:
		return int(res.StatusCode), nil
	case err := <-waiting.Error:
		return 0, fmt.Errorf("wait for container: %w", err)
	case <-ctx.Done():
		return 0, fmt.Errorf("stopped: %w", ctx.Err())
	}
}

func (d *Docker) oomKilled(ctx context.Context, id string) bool {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	res, err := d.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil || res.Container.State == nil {
		return false
	}

	return res.Container.State.OOMKilled
}

func (d *Docker) createOptions(spec Spec, mounts []mount.Mount) client.ContainerCreateOptions {
	limits, enabled := d.cfg.Limits, true
	pids, memory, file := signed(limits.Processes), signed(limits.MemoryBytes), signed(limits.FileBytes)
	fsize := &container.Ulimit{Name: "fsize", Soft: file, Hard: file}
	// With the network flag set Docker writes no /etc/hosts, so a database name could not be given its local
	// address. NetworkMode "none" below still leaves the container with only a loopback interface.
	hosts := tunnelHosts(spec)

	return client.ContainerCreateOptions{
		Name: d.containerName(spec),
		Config: &container.Config{
			Image: d.cfg.Image, Cmd: command(spec), Env: containerEnv(spec), WorkingDir: Workspace,
			User:   strconv.Itoa(d.cfg.UID) + ":" + strconv.Itoa(d.cfg.GID),
			Labels: map[string]string{taskLabel: d.cfg.Owner}, NetworkDisabled: len(hosts) == 0,
			AttachStdin: spec.Stdin != nil, OpenStdin: spec.Stdin != nil, StdinOnce: spec.Stdin != nil,
			AttachStdout: true, AttachStderr: true,
			// The image's own entrypoint must not wrap the command.
			Entrypoint: []string{},
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "none", ReadonlyRootfs: true, Init: &enabled, Runtime: d.cfg.Runtime,
			ExtraHosts: hosts,
			CapDrop:    []string{"ALL"}, SecurityOpt: []string{"no-new-privileges"},
			Mounts: mounts, Tmpfs: map[string]string{"/tmp": tmpfsOpts},
			Resources: container.Resources{
				Memory: memory, MemorySwap: memory + signed(limits.SwapBytes), PidsLimit: &pids,
				NanoCPUs: signed(limits.CPUMillis) * nanoPerMS, Ulimits: []*container.Ulimit{fsize},
			},
		},
	}
}

const buildStageCommand = `set -euo pipefail
stage=/output/.build-stage
trap 'find "$stage" -mindepth 1 -delete 2>/dev/null || true' EXIT HUP INT TERM
cd /workspace || { echo 'Build staging failed: source workspace is unavailable.' >&2; exit 125; }
if ! find -P . -xdev \
  \( -path './.git' -o -name '.workspace' \
     -o -path './.build-output-*' -o -path './.artifact-output-*' \
     -o -name '.env' -o -name '.env.*' -o -name '.npmrc' -o -name '.pypirc' \
     -o -name '.netrc' -o -name '.ssh' -o -name '.docker' \
     -o -iname '*credentials*' -o -iname '*secret*' -o -name 'id_rsa*' -o -name 'id_ed25519*' \
     -o -name '*.token' \
     -o -name '*.pem' -o -name '*.key' -o -name '*.p12' -o -name '*.pfx' \
     -o -name '*.jks' \) -prune -o \( -type d -o -type f \) -print0 |
  tar --no-recursion --null --files-from=- -cf - |
  tar --no-same-owner --no-same-permissions -C "$stage" -xf -; then
  echo 'Build staging failed: could not copy eligible project files.' >&2
  exit 125
fi
cd "$stage" || { echo 'Build staging failed: scratch directory is unavailable.' >&2; exit 125; }
"$@"`

// command preserves the image proxy wrapper and stages only explicitly named builds.
func command(spec Spec) []string {
	argv := spec.Argv
	if spec.BuildStage {
		argv = append([]string{"bash", "-c", buildStageCommand, "build-stage"}, argv...)
	}
	if spec.Egress == "" && spec.Site == "" && spec.View == "" && len(spec.Tunnels) == 0 {
		return argv
	}

	return append([]string{Entry}, argv...)
}

func (d *Docker) containerName(spec Spec) string {
	role := "agentic-sandbox-"
	if spec.Site != "" {
		role = "agentic-site-"
	}

	owner := strings.ToLower(d.cfg.Owner)
	if len(owner) > ownerTagLen {
		owner = owner[:ownerTagLen]
	}

	return fmt.Sprintf("%s%s-%d", role, owner, containerSeq.Add(1))
}

// signed converts a limit for the Docker API, which counts in int64.
func signed(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(v)
}
