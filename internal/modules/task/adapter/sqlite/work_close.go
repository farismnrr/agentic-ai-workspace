package sqlite

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func closesWork(change application.WorkChange) bool {
	if change.Patch == nil || change.Patch.State == nil {
		return false
	}
	switch *change.Patch.State {
	case domain.WorkCompleted, domain.WorkFailed, domain.WorkStopped, domain.WorkAbandoned:
		return true
	case domain.WorkOpen, domain.WorkBlocked:
		return false
	default:
		return false
	}
}
