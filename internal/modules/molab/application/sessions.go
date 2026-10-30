package application

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ErrUnavailable = errors.New("no active MoLab pairing")
var ErrInvalid = errors.New("invalid MoLab session")

const SessionTTL = 30 * time.Minute
const MaxSessions = 200

var hostPattern = regexp.MustCompile(`^sb-[a-z0-9-]+\.sb\.molab\.run$`)

type Session struct {
	UserID    string
	ProjectID string
	Inspected bool
	URL       string
	Token     string
	ExpiresAt time.Time
	ctx       context.Context
	cancel    context.CancelFunc
}
type Status struct {
	Connected bool       `json:"connected"`
	URL       string     `json:"url,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
type Sessions struct {
	mu     sync.Mutex
	values map[string]Session
	now    func() time.Time
}

func NewSessions(now func() time.Time) *Sessions {
	if now == nil {
		now = time.Now
	}
	return &Sessions{values: make(map[string]Session), now: now}
}
func Validate(raw, token string) (string, error) {
	if len(raw) > 512 || len(token) < 8 || len(token) > 512 || strings.ContainsAny(token, "\r\n\t ") {
		return "", ErrInvalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		return "", ErrInvalid
	}
	if u.Port() != "" && u.Port() != "443" {
		return "", ErrInvalid
	}
	host := strings.ToLower(u.Hostname())
	if !hostPattern.MatchString(host) {
		return "", ErrInvalid
	}
	return "https://" + host + "/", nil
}
func (s *Sessions) Put(task, user, project, raw, token string) (Status, error) {
	u, e := Validate(raw, token)
	if e != nil {
		return Status{}, e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	if len(s.values) >= MaxSessions && s.values[task].UserID == "" {
		return Status{}, ErrUnavailable
	}
	if previous, ok := s.values[task]; ok {
		previous.cancel()
	}
	live, cancel := context.WithCancel(context.Background())
	v := Session{UserID: user, ProjectID: project, URL: u, Token: token, ExpiresAt: s.now().Add(SessionTTL), ctx: live, cancel: cancel}
	s.values[task] = v
	return Status{Connected: true, URL: u, ExpiresAt: &v.ExpiresAt}, nil
}
func (s *Sessions) Get(task, user, project string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	v, ok := s.values[task]
	if !ok || v.UserID != user || v.ProjectID != project {
		return Session{}, ErrUnavailable
	}
	return v, nil
}
func (s *Sessions) Status(task, user, project string) Status {
	v, e := s.Get(task, user, project)
	if e != nil {
		return Status{}
	}
	return Status{Connected: true, URL: v.URL, ExpiresAt: &v.ExpiresAt}
}

// MarkInspected only changes the same pairing instance, not a replacement with a reused token.
func (s *Sessions) MarkInspected(task string, used Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[task]
	if ok && v.ctx == used.ctx {
		v.Inspected = true
		s.values[task] = v
	}
}
func (s *Sessions) Forget(task string) {
	s.mu.Lock()
	if prior, ok := s.values[task]; ok {
		prior.cancel()
	}
	delete(s.values, task)
	s.mu.Unlock()
}

// Bound cancels an in-flight execution on disconnect/replacement or session expiry.
func (session Session) Bound(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(parent, session.ExpiresAt)
	go func() {
		select {
		case <-session.ctx.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
func (s *Sessions) prune() {
	for k, v := range s.values {
		if !s.now().Before(v.ExpiresAt) {
			v.cancel()
			delete(s.values, k)
		}
	}
}

func (s *Sessions) SweepUntil(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.Clear()
			return
		case <-ticker.C:
			s.mu.Lock()
			s.prune()
			s.mu.Unlock()
		}
	}
}
func (s *Sessions) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.values {
		v.cancel()
		delete(s.values, k)
	}
}
