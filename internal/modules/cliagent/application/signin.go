package application

import (
	"context"
	"io"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

const (
	// signInWindow is how long a person has to open the address and finish.
	signInWindow = 15 * time.Minute
	// promptWait is how long the program may take to print the address.
	promptWait = 45 * time.Second
	// codeWait is how long the program may take to accept a pasted code.
	codeWait = 90 * time.Second
)

// signIn is one program waiting for a person to finish in their browser.
type signIn struct {
	scan   *scanner
	stdin  *io.PipeWriter
	cancel context.CancelFunc
	done   chan struct{}
	// ok is set before done is closed.
	ok bool
}

// begin starts the program's sign-in, or returns the one already running for this person.
func (s *Service) begin(ctx context.Context, user string, rt domain.Runtime) *signIn {
	k := key{user, rt}

	s.mu.Lock()
	defer s.mu.Unlock()

	if current := s.active[k]; current != nil {
		return current
	}

	prog := programOf(rt)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), signInWindow)
	si := &signIn{scan: newScanner(rt), cancel: cancel, done: make(chan struct{})}
	job := Job{UserID: user, Argv: prog.login, Env: prog.env, Hosts: s.hosts[rt]}

	if rt == domain.Claude {
		reader, writer := io.Pipe()
		job.Stdin, si.stdin = reader, writer
	}

	s.active[k] = si

	go s.run(ctx, k, si, job)

	return si
}

// run keeps the program going until it ends, then forgets the sign-in.
func (s *Service) run(ctx context.Context, k key, si *signIn, job Job) {
	defer si.cancel()

	exit, err := s.sandbox.Run(ctx, job, si.scan)
	if si.stdin != nil {
		_ = si.stdin.Close()
	}

	si.ok = err == nil && exit == 0

	s.mu.Lock()

	if s.active[k] == si {
		delete(s.active, k)
	}

	delete(s.checked, k)
	s.mu.Unlock()

	close(si.done)
}

func (s *Service) current(user string, rt domain.Runtime) *signIn {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.active[key{user, rt}]
}

// deliver hands the pasted code to the program.
func deliver(ctx context.Context, si *signIn, code string) error {
	wrote := make(chan error, 1)

	go func() {
		_, err := io.WriteString(si.stdin, code+"\n")
		wrote <- err
	}()

	select {
	case err := <-wrote:
		if err != nil {
			return domain.ErrFailed
		}

		return nil
	case <-si.done:
		return domain.ErrFailed
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the caller's own cancellation
	}
}

// outcome waits for the program to say whether the code was accepted.
func outcome(ctx context.Context, si *signIn) error {
	wait := time.NewTimer(codeWait)
	defer wait.Stop()

	select {
	case <-si.done:
		if si.ok {
			return nil
		}

		return domain.ErrFailed
	case <-wait.C:
		si.cancel()

		return domain.ErrFailed
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the caller's own cancellation
	}
}
