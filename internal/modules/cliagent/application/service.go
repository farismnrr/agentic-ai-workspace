package application

import (
	"context"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

const (
	checkWait = 30 * time.Second
	statusTTL = 30 * time.Second
)

var codeShape = regexp.MustCompile(`^[A-Za-z0-9._~#%+/=:-]{8,512}$`)

type key struct {
	user string
	rt   domain.Runtime
}

type checked struct {
	at        time.Time
	connected bool
}

// Service lets a person sign in to a vendor's coding program with their own plan, and tells
// whether they are signed in. It never reads the sign-in itself: only the program does.
type Service struct {
	sandbox  Sandbox
	hosts    map[domain.Runtime][]string
	now      func() time.Time
	active   map[key]*signIn
	checked  map[key]checked
	catalogs map[key]catalog
	mu       sync.Mutex
}

// NewService builds the service; hosts are the addresses each program may reach.
func NewService(sandbox Sandbox, hosts map[domain.Runtime][]string, now func() time.Time) *Service {
	return &Service{
		sandbox: sandbox, hosts: hosts, now: now,
		active: map[key]*signIn{}, checked: map[key]checked{}, catalogs: map[key]catalog{},
	}
}

// Status says whether the person is signed in, or what they must do to finish signing in.
func (s *Service) Status(ctx context.Context, user string, rt domain.Runtime) (domain.State, error) {
	k := key{user, rt}

	s.mu.Lock()

	if si := s.active[k]; si != nil {
		s.mu.Unlock()

		return waiting(si), nil
	}

	if c, ok := s.checked[k]; ok && s.now().Sub(c.at) < statusTTL {
		s.mu.Unlock()

		return domain.State{Connected: c.connected}, nil
	}

	s.mu.Unlock()

	connected, err := s.check(ctx, k)
	if err != nil {
		return domain.State{}, err
	}

	return domain.State{Connected: connected}, nil
}

func waiting(si *signIn) domain.State {
	select {
	case <-si.scan.found:
		p := si.scan.Prompt()

		return domain.State{Prompt: &p}
	default:
		return domain.State{}
	}
}

// check asks the program itself; a zero exit code means it holds a working sign-in.
func (s *Service) check(ctx context.Context, k key) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, checkWait)
	defer cancel()

	prog := programOf(k.rt)
	exit, err := s.sandbox.Run(ctx, Job{UserID: k.user, Argv: prog.status, Env: prog.env}, io.Discard)
	if err != nil {
		return false, domain.ErrUnavailable
	}

	s.mu.Lock()
	s.checked[k] = checked{at: s.now(), connected: exit == 0}
	s.mu.Unlock()

	return exit == 0, nil
}

// Start begins signing in and returns what the person must open in their browser.
func (s *Service) Start(ctx context.Context, user string, rt domain.Runtime) (domain.Prompt, error) {
	si := s.begin(ctx, user, rt)
	wait := time.NewTimer(promptWait)

	defer wait.Stop()

	select {
	case <-si.scan.found:
		return si.scan.Prompt(), nil
	case <-si.done:
		return domain.Prompt{}, domain.ErrFailed
	case <-wait.C:
		si.cancel()

		return domain.Prompt{}, domain.ErrFailed
	case <-ctx.Done():
		si.cancel()

		return domain.Prompt{}, ctx.Err() //nolint:wrapcheck // the caller's own cancellation
	}
}

// Submit gives the code the vendor showed the person to the waiting program.
func (s *Service) Submit(ctx context.Context, user string, rt domain.Runtime, code string) error {
	code = strings.TrimSpace(code)
	if !codeShape.MatchString(code) {
		return domain.ErrBadCode
	}

	si := s.current(user, rt)
	if si == nil || si.stdin == nil {
		return domain.ErrNoSignIn
	}

	if err := deliver(ctx, si, code); err != nil {
		return err
	}

	return outcome(ctx, si)
}

// Disconnect stops any sign-in and has the program forget the person's plan.
func (s *Service) Disconnect(ctx context.Context, user string, rt domain.Runtime) error {
	if si := s.current(user, rt); si != nil {
		si.cancel()
		<-si.done
	}

	prog := programOf(rt)
	if _, err := s.sandbox.Run(ctx, Job{UserID: user, Argv: prog.logout, Env: prog.env}, io.Discard); err != nil {
		return domain.ErrUnavailable
	}

	s.mu.Lock()
	delete(s.checked, key{user, rt})
	delete(s.catalogs, key{user, rt})
	s.mu.Unlock()

	return nil
}
