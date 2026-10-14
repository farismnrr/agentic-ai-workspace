//go:build unix

package runner

import (
	"os/exec"
	"syscall"
)

// isolate gives the task its own process group so a timeout kills its children too.
func isolate(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
