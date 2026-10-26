//go:build !unix

// Package instancelock lets one process own a file for as long as it runs.
package instancelock

import "errors"

// ErrHeld means another process already holds the lock.
var ErrHeld = errors.New("another instance already holds the lock")

var errUnsupported = errors.New("instance locks are not supported on this system")

// Acquire is not available here, so features that need it stay off.
func Acquire(string) (func(), error) { return nil, errUnsupported }
