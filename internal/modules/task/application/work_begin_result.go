package application

import (
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// BeginResult confirms the created or resumed work and the available roles.
//
//nolint:govet // fieldalignment: its compact internal mode preserves the required JSON response shape.
type BeginResult struct {
	mode        beginMode
	TaskID      string
	WorkID      string
	Roles       []WorkRole
	Resumed     bool
	Idempotency string
}

func (r BeginResult) MarshalJSON() ([]byte, error) {
	idempotency := "none"
	if r.mode != beginWithoutKey {
		idempotency = "keyed"
	}
	encoded, err := json.Marshal(beginResponse{
		Roles: r.Roles, WorkID: r.WorkID, Idempotency: idempotency, Resumed: r.mode == beginResumed,
	})
	if err != nil {
		return nil, fmt.Errorf("encode begin response: %w", err)
	}
	return encoded, nil
}

//nolint:govet // fieldalignment: this JSON response mirrors the public tool contract.
type beginResponse struct {
	Roles       []WorkRole `json:"roles"`
	WorkID      string     `json:"workId"`
	Idempotency string     `json:"idempotency"`
	Resumed     bool       `json:"resumed"`
}

func (w *Works) beginResult(work domain.ExternalWork, resumed bool, roster Roster) BeginResult {
	idempotency := "none"
	if work.IdemKey != "" {
		idempotency = "keyed"
	}
	mode := beginWithoutKey
	if idempotency == "keyed" {
		mode = beginWithKey
	}
	if resumed {
		mode = beginResumed
	}
	return BeginResult{
		mode: mode, TaskID: work.TaskID, WorkID: work.ID, Roles: publicRoles(roster),
		Resumed: resumed, Idempotency: idempotency,
	}
}
