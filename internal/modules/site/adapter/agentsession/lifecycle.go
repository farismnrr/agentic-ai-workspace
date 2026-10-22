package agentsession

import (
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func (r *Registry) sweep() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	defer close(r.done)
	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			r.expireLocked(r.now())
			r.mu.Unlock()
		case <-r.stop:
			return
		}
	}
}

// RevokeTask removes every container session owned by the task.
func (r *Registry) RevokeTask(userID, taskID string) {
	r.revokeMatching(func(scope application.AgentSessionScope) bool {
		return scope.UserID == userID && scope.TaskID == taskID
	})
}

// Reset drops one scoped jar and cancels any operation using it.
func (r *Registry) Reset(scope application.AgentSessionScope) {
	r.revokeMatching(func(candidate application.AgentSessionScope) bool { return candidate == scope })
}

// RevokeProject removes every session belonging to a site's project.
func (r *Registry) RevokeProject(projectID string) {
	r.revokeMatching(func(scope application.AgentSessionScope) bool { return scope.ProjectID == projectID })
}

// RevokeUser removes every session belonging to a person.
func (r *Registry) RevokeUser(userID string) {
	r.revokeMatching(func(scope application.AgentSessionScope) bool { return scope.UserID == userID })
}

func (r *Registry) revokeMatching(matches func(application.AgentSessionScope) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for scope := range r.entries {
		if matches(scope) {
			r.revokeLocked(scope)
		}
	}
}

// Close cancels all operations, drops every jar and stops expiry cleanup.
func (r *Registry) Close() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		for scope := range r.entries {
			r.revokeLocked(scope)
		}
		close(r.stop)
	}
	r.mu.Unlock()
	r.stopped.Do(func() { <-r.done })
}

func valid(scope application.AgentSessionScope) bool {
	return scope.UserID != "" && scope.ProjectID != "" && scope.TaskID != "" && scope.ContainerID != ""
}

func (r *Registry) expireLocked(now time.Time) {
	for scope, item := range r.entries {
		lastUsed := time.Unix(0, item.lastUsed)
		createdAt := time.Unix(0, item.createdAt)
		if now.Sub(lastUsed) >= r.idle || now.Sub(createdAt) >= r.absolute {
			r.revokeLocked(scope)
		}
	}
}

func (r *Registry) revokeLocked(scope application.AgentSessionScope) {
	if item := r.entries[scope]; item != nil {
		delete(r.entries, scope)
		close(item.done)
	}
}
