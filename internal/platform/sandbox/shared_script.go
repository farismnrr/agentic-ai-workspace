package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

const (
	bytesPerBlock = 1024
	loopbackUpPy  = `import socket,fcntl,struct;` +
		`s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);` +
		`fcntl.ioctl(s,0x8914,struct.pack("16sH",b"lo",0x49))`
)

func (d *Docker) sharedCmd(runID string, spec Spec, mounts []mount.Mount, logFile string) []string {
	var inner strings.Builder
	inner.WriteString("set -e\n")
	inner.WriteString("python3 -c '" + loopbackUpPy + "'\n")
	inner.WriteString("mount -t tmpfs -o " + tmpfsOpts + " tmpfs /tmp\n")

	for _, m := range mounts {
		src := d.sharedMountSource(m)
		inner.WriteString("mount --bind " + shellQuote(src) + " " + shellQuote(m.Target) + "\n")
		if m.ReadOnly {
			inner.WriteString("mount -o remount,ro,bind " + shellQuote(m.Target) + "\n")
		}
	}

	if hosts := tunnelHosts(spec); len(hosts) > 0 {
		inner.WriteString("cp /etc/hosts /tmp/.hosts\n")
		for _, entry := range hosts {
			host, ip, _ := strings.Cut(entry, ":")
			inner.WriteString("printf '%s\\t%s\\n' " + shellQuote(ip) + " " + shellQuote(host) + " >> /tmp/.hosts\n")
		}
		inner.WriteString("mount --bind /tmp/.hosts /etc/hosts\n")
		inner.WriteString("mount -o remount,ro,bind /etc/hosts\n")
		inner.WriteString("rm -f /tmp/.hosts\n")
	}

	inner.WriteString("mount -t tmpfs -o " + maskTmpfsOpts + " tmpfs /mnt\n")
	inner.WriteString("mount -t tmpfs -o " + maskTmpfsOpts + " tmpfs " + sharedPidsDir + "\n")
	inner.WriteString("cd " + Workspace + "\n")

	if d.cfg.Limits.FileBytes > 0 {
		blocks := d.cfg.Limits.FileBytes / bytesPerBlock
		if blocks > 0 {
			limit := strconv.FormatUint(blocks, 10)
			inner.WriteString("ulimit -Sf " + limit + " -Hf " + limit + " 2>/dev/null || true\n")
		}
	}

	uid := strconv.Itoa(d.cfg.UID)
	gid := strconv.Itoa(d.cfg.GID)
	inner.WriteString("exec setpriv --reuid=" + uid + " --regid=" + gid +
		" --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs -- \"$@\"")

	pidFile := path.Join(sharedPidsDir, runID+".pid")
	outer := "echo $$ > " + shellQuote(pidFile) + "\n"
	if logFile != "" {
		outer += "exec >" + shellQuote(logFile) + " 2>&1\n"
	}
	outer += "exec unshare --mount --net --ipc --pid --fork sh -c " + shellQuote(inner.String()) + " -- \"$@\""

	return append([]string{"sh", "-c", outer, "--"}, command(spec)...)
}

func (d *Docker) sharedMountSource(m mount.Mount) string {
	sub := ""
	if m.VolumeOptions != nil {
		sub = m.VolumeOptions.Subpath
	}

	return path.Join(sharedVolumesRoot, m.Source, sub)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (d *Docker) startShared(ctx context.Context, spec Spec, mounts []mount.Mount) (string, error) {
	d.ensureSharedSandbox(ctx)

	id := fmt.Sprintf("%s%s-%d", sharedSitePrefix, strings.ToLower(d.cfg.Owner), containerSeq.Add(1))
	logFile := path.Join(sharedPidsDir, id+".log")

	created, err := d.api.ExecCreate(ctx, SharedSandboxName, client.ExecCreateOptions{
		User:       rootUser,
		Env:        containerEnv(spec),
		WorkingDir: Workspace,
		Cmd:        d.sharedCmd(id, spec, mounts, logFile),
	})
	if err != nil {
		return "", fmt.Errorf("create site exec: %w", err)
	}

	if _, err = d.api.ExecStart(ctx, created.ID, client.ExecStartOptions{Detach: true}); err != nil {
		d.killShared(ctx, id)
		return "", fmt.Errorf("start site exec: %w", err)
	}

	return id, nil
}

func (d *Docker) killShared(ctx context.Context, id string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	pidFile := path.Join(sharedPidsDir, id+".pid")
	logFile := path.Join(sharedPidsDir, id+".log")
	script := "if [ -f " + shellQuote(pidFile) + " ]; then " +
		"pid=$(cat " + shellQuote(pidFile) + "); " +
		"pkill -9 -P \"$pid\" 2>/dev/null || true; " +
		"kill -9 \"$pid\" 2>/dev/null || true; " +
		"rm -f " + shellQuote(pidFile) + "; fi; " +
		"rm -f " + shellQuote(logFile)

	created, err := d.api.ExecCreate(ctx, SharedSandboxName, client.ExecCreateOptions{
		User: rootUser, Cmd: []string{"sh", "-c", script},
	})
	if err == nil {
		_, _ = d.api.ExecStart(ctx, created.ID, client.ExecStartOptions{Detach: true})
	}
}

func (d *Docker) runningShared(ctx context.Context, id string) bool {
	pidFile := path.Join(sharedPidsDir, id+".pid")
	script := "test -f " + shellQuote(pidFile) + " && kill -0 $(cat " + shellQuote(pidFile) + ") 2>/dev/null"
	out, code := d.execOutput(ctx, script)

	return code == 0 && out == ""
}

func (d *Docker) logTailShared(ctx context.Context, id, lines string) string {
	logFile := path.Join(sharedPidsDir, id+".log")
	script := "tail -n " + shellQuote(lines) + " " + shellQuote(logFile) + " 2>/dev/null || true"
	out, _ := d.execOutput(ctx, script)

	return out
}

func (d *Docker) execOutput(ctx context.Context, script string) (string, int) {
	created, err := d.api.ExecCreate(ctx, SharedSandboxName, client.ExecCreateOptions{
		User: rootUser, AttachStdout: true, AttachStderr: true, Cmd: []string{"sh", "-c", script},
	})
	if err != nil {
		return "", 1
	}

	attached, err := d.api.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", 1
	}
	defer attached.Close()

	var out bytes.Buffer
	_, _ = stdcopy.StdCopy(&out, &out, attached.Reader)

	inspected, err := d.api.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return out.String(), 1
	}

	return out.String(), inspected.ExitCode
}
