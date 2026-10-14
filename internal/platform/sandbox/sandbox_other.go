//go:build !linux

package sandbox

import (
	"context"
	"os/exec"
)

// Command is unavailable off Linux.
func Command(context.Context, Spec) (*exec.Cmd, func(), error) {
	return nil, nil, ErrUnsupported
}

// RunIfChild does nothing off Linux.
func RunIfChild() {}

// Group is a cgroup; there are none off Linux.
type Group struct{}

// Cgroups hands out cgroups; there are none off Linux.
type Cgroups struct{}

// OpenCgroups is unavailable off Linux.
func OpenCgroups() (*Cgroups, error) { return nil, ErrUnsupported }

// New is unavailable off Linux.
func (*Cgroups) New(Limits) (*Group, error) { return nil, ErrUnsupported }

// FD is meaningless off Linux.
func (*Group) FD() int { return -1 }

// Close does nothing off Linux.
func (*Group) Close() {}

// OOMKilled is always false off Linux.
func (*Group) OOMKilled() bool { return false }

// Install is unavailable off Linux.
func Install(context.Context, InstallOptions) error { return ErrUnsupported }
