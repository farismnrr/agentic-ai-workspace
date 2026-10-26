package egressproxy

import (
	"context"
	"errors"
	"slices"
	"strings"
)

// ErrGrantsOff means the hub was given no grant registry, which is how the feature stays off.
var ErrGrantsOff = errors.New("dynamic egress grants are not enabled")

// WithGrants lets the hub serve task sockets backed by g. Without it TaskDir refuses, and nothing else changes.
func (h *Hub) WithGrants(g *Grants) *Hub {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.grants = g

	return h
}

// TaskDir returns the folder of the socket for one task, serving it on first use. The socket answers exactly like
// a workspace socket (the static list and any extra hosts, which are the caller's own and go through WithHosts as
// before) and, for hosts those do not allow, consults the grants of this scope and no other. The scope is fixed here,
// by the server, when the listener is made: it is not read from anything a client sends. Every scope has its own
// socket and folder, and a sandbox is only ever handed its own.
func (h *Hub) TaskDir(ctx context.Context, id string, scope Scope, extra ...string) (string, error) {
	h.mu.Lock()
	grants := h.grants
	h.mu.Unlock()

	if grants == nil {
		return "", ErrGrantsOff
	}

	if !scope.valid() || id == "" {
		return "", ErrBadGrant
	}

	hosts := strings.Join(slices.Sorted(slices.Values(extra)), ",")
	key := strings.Join([]string{"task", id, scope.Project, scope.Task, hosts}, "\x00")

	return h.serve(ctx, key, served{id: id, task: scope}, func() *Proxy {
		proxy := h.proxy.WithHosts(extra)
		proxy.task.grants, proxy.task.scope = grants, scope

		return proxy
	})
}

// RemoveTask stops serving one task's sockets and revokes its grants, for when the task is over.
func (h *Hub) RemoveTask(scope Scope) {
	if !scope.valid() {
		return
	}

	h.mu.Lock()

	for key, s := range h.servers {
		if s.task == scope {
			h.stop(key, s)
		}
	}

	grants := h.grants
	h.mu.Unlock()

	if grants != nil {
		grants.Clear(scope)
	}
}
