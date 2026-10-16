package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// AgentRole is one member of the agent team, as the loop needs to know it.
type AgentRole struct {
	Key     string
	Name    string
	Purpose string
	Prompt  string
	// Skills are the role's built-in playbooks: the ones marked Always are put in its prompt, the others are
	// listed and loaded when the job needs them.
	Skills []Skill
	// Library lists what the role can look up while it works, and Lookup reads one item of it.
	Library    []Reference
	Lookup     func(name string) (string, bool)
	ProviderID string
	Model      string
	// Effort is empty when the role has none of its own.
	Effort domain.Effort
	// Tools names the project tools the role may use; the lead is not limited by it.
	Tools []string
	Lead  bool
}

// Skill is one playbook of a role: what it is for, and the instructions themselves.
type Skill struct {
	Name        string
	Description string
	Body        string
	// Craft names the library items that are loaded together with the skill.
	Craft  []string
	Always bool
}

// Reference is one item a role can look up, such as a style family or a craft rule.
type Reference struct {
	Name    string
	Summary string
}

// Roster is the team on one task: the orchestrator and the teammates it may hand work to.
type Roster struct {
	Members []AgentRole
	Lead    AgentRole
}

func (r Roster) member(key string) (AgentRole, bool) {
	for _, member := range r.Members {
		if member.Key == key {
			return member, true
		}
	}

	return AgentRole{}, false
}

// Team gives the loop a person's agent team.
type Team interface {
	// Roster returns the lead and the enabled teammates.
	Roster(ctx context.Context, userID string) (Roster, error)
	// UseByDefault says whether the person wants new tasks to start with the team.
	UseByDefault(ctx context.Context, userID string) bool
}

// ScopedTranscripts keeps a teammate's conversation apart from the orchestrator's.
type ScopedTranscripts interface {
	// Scope returns the transcript of one delegation; the empty scope is the orchestrator's own.
	Scope(scope string) TranscriptStore
}

// Delegation statuses.
const (
	DelegationWorking = "working"
	DelegationWaiting = "waiting"
	DelegationDone    = "done"
	DelegationFailed  = "failed"
)

// Delegation is one job the orchestrator handed to a teammate. Its id is the delegate tool call's id.
type Delegation struct {
	ID         string
	RoleKey    string
	RoleName   string
	ProviderID string
	Model      string
	Effort     domain.Effort
	Brief      string
	Status     string
	Report     string
	// StartSeq is the last thread event before the teammate began, so their steps can be found again.
	StartSeq int64
}

// DelegationStore persists delegations, so a paused teammate resumes with the same model and brief.
type DelegationStore interface {
	StartDelegation(ctx context.Context, taskID string, d Delegation, at time.Time) error
	FindDelegation(ctx context.Context, taskID, id string) (Delegation, bool, error)
	SetDelegationStatus(ctx context.Context, taskID, id, status string) error
	EndDelegation(ctx context.Context, taskID, id, status, report string, at time.Time) error
}
