package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

// ChangedSinceCommit lets the external tests ask the question without a whole run.
func ChangedSinceCommit(events []domain.Event) bool { return changedSinceCommit(events) }
