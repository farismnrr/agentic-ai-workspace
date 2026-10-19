// Package application starts a vendor's own coding program for a person, so they sign in with
// their own plan. The server never sees the sign-in: the program keeps it in the person's folder.
package application

import (
	"context"
	"io"
)

// Job is one run of a vendor's program.
type Job struct {
	// Stdin, when set, is what the program reads from its input.
	Stdin io.Reader
	// UserID is the person whose folder the program runs in.
	UserID string
	Argv   []string
	Env    []string
	// Hosts are the only addresses the program can reach.
	Hosts []string
}

// Sandbox runs a program in an isolated container with the person's own folder.
type Sandbox interface {
	// Run waits for the program to end and returns its exit code. out gets everything it prints.
	Run(ctx context.Context, job Job, out io.Writer) (exit int, err error)
}
