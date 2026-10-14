//go:build !unix

package runner

import "os/exec"

// isolate has nothing to add where process groups are unavailable.
func isolate(*exec.Cmd) {}
