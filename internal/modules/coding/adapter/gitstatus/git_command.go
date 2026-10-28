package gitstatus

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const commandErrorOutputLimit = 4096

func runGit(snapshot *metadataSnapshot, limit int, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	output := &prefixBuffer{limit: limit}
	// The executable is fixed and the argv is assembled only from this package's
	// fixed Git operations plus validated path/object values. No shell parses it.
	cmd := exec.CommandContext(ctx, "git")
	cmd.Args = append(cmd.Args, gitCommandArgs(args...)...)
	cmd.Env = gitEnvironment(snapshot)
	cmd.Stdout = output
	cmd.Stderr = &prefixBuffer{limit: commandErrorOutputLimit}
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", errGitCommandTimeout, err)
		}
		return nil, fmt.Errorf("execute fixed Git command: %w", err)
	}
	return output.Bytes(), nil
}

func gitCommandArgs(args ...string) []string {
	commandArgs := []string{
		"-c", "core.bare=false", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null",
		"-c", "core.attributesFile=/dev/null", "-c", "core.excludesFile=/dev/null",
	}
	return append(commandArgs, args...)
}

func gitEnvironment(snapshot *metadataSnapshot) []string {
	return []string{
		"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_DIR=" + snapshot.gitDir, "GIT_WORK_TREE=" + snapshot.worktree,
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_ATTR_NOSYSTEM=1",
		"GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0",
	}
}

type prefixBuffer struct {
	bytes.Buffer
	limit int
}

func (b *prefixBuffer) Write(p []byte) (int, error) {
	written := len(p)
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	_, _ = b.Buffer.Write(p)
	return written, nil
}
