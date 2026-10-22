// Package agentsession stores short-lived screenshot cookies in memory.
package agentsession

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

const (
	defaultCapacity = 256
	defaultUserCap  = 16
	idleLifetime    = 30 * time.Minute
	absoluteLife    = 4 * time.Hour
	operationLimit  = 90 * time.Second
	cleanupInterval = time.Minute
)

var (
	ErrInvalidScope = errors.New("invalid agent session scope")
	ErrFull         = errors.New("agent session capacity reached")
	ErrClosed       = errors.New("agent session registry closed")
)

type entry struct {
	jar       *boundedJar
	gate      chan struct{}
	done      chan struct{}
	scope     application.AgentSessionScope
	createdAt int64
	lastUsed  int64
}

// Registry bounds session state and serializes each session's operations.
type Registry struct {
	entries  map[application.AgentSessionScope]*entry
	now      func() time.Time
	done     chan struct{}
	stop     chan struct{}
	capacity int
	userCap  int
	idle     time.Duration
	absolute time.Duration
	limit    time.Duration
	closed   bool
	mu       sync.Mutex
	stopped  sync.Once
}

// New starts a bounded registry with production lifetime limits.
func New() *Registry {
	return NewWithLimits(defaultCapacity, idleLifetime, absoluteLife, operationLimit, time.Now)
}

// NewWithLimits permits deterministic lifecycle tests.
func NewWithLimits(capacity int, idle, absolute, operation time.Duration, now func() time.Time) *Registry {
	return NewWithUserLimit(capacity, min(capacity, defaultUserCap), idle, absolute, operation, now)
}

// NewWithUserLimit permits deterministic tests of global and per-user fairness limits.
func NewWithUserLimit(
	capacity, userCap int, idle, absolute, operation time.Duration, now func() time.Time,
) *Registry {
	if capacity < 1 || userCap < 1 || userCap > capacity || idle <= 0 || absolute <= 0 || operation <= 0 || now == nil {
		panic("invalid agent session registry limits")
	}
	r := &Registry{
		entries: make(map[application.AgentSessionScope]*entry), capacity: capacity,
		userCap: userCap,
		idle:    idle, absolute: absolute, limit: operation, now: now,
		done: make(chan struct{}), stop: make(chan struct{}),
	}
	go r.sweep()
	return r
}

// Do creates or reuses one scoped cookie jar and allows one cancelable operation at a time.
func (r *Registry) Do(ctx context.Context, scope application.AgentSessionScope,
	operation func(context.Context, http.CookieJar) error,
) error {
	if !valid(scope) || operation == nil {
		return ErrInvalidScope
	}
	item, err := r.get(scope)
	if err != nil {
		return err
	}
	select {
	case item.gate <- struct{}{}:
		defer func() { <-item.gate }()
	case <-ctx.Done():
		return fmt.Errorf("wait for agent session: %w", ctx.Err())
	case <-item.done:
		return context.Canceled
	}
	if !r.current(scope, item) {
		return context.Canceled
	}
	return r.run(ctx, item, operation)
}

func (r *Registry) run(ctx context.Context, item *entry,
	operation func(context.Context, http.CookieJar) error,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.limit)
	defer cancel()
	ended := make(chan struct{})
	defer close(ended)
	go func() {
		select {
		case <-item.done:
			cancel()
		case <-ended:
		}
	}()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("check agent session context: %w", err)
	}
	r.mu.Lock()
	item.lastUsed = r.now().UnixNano()
	r.mu.Unlock()
	if err := operation(ctx, item.jar); err != nil {
		return fmt.Errorf("agent session operation: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("agent session operation context: %w", err)
	}
	return nil
}

func (r *Registry) get(scope application.AgentSessionScope) (*entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrClosed
	}
	r.expireLocked(r.now())
	if item := r.entries[scope]; item != nil {
		return item, nil
	}
	if len(r.entries) >= r.capacity {
		return nil, ErrFull
	}
	if r.countUserLocked(scope.UserID) >= r.userCap {
		return nil, ErrFull
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	now := r.now()
	bounded := &boundedJar{jar: jar, keys: make(map[string]cookieRecord)}
	item := &entry{
		jar:       bounded,
		gate:      make(chan struct{}, 1),
		done:      make(chan struct{}),
		scope:     scope,
		createdAt: now.UnixNano(),
		lastUsed:  now.UnixNano(),
	}
	r.entries[scope] = item
	return item, nil
}

func (r *Registry) countUserLocked(userID string) int {
	count := 0
	for scope := range r.entries {
		if scope.UserID == userID {
			count++
		}
	}
	return count
}

func (r *Registry) current(scope application.AgentSessionScope, item *entry) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expireLocked(r.now())
	return !r.closed && r.entries[scope] == item
}
