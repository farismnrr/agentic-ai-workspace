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
	mode   beginMode
	Roles  []WorkRole
	WorkID string
}

func (r BeginResult) MarshalJSON() ([]byte, error) {
	mode := "none"
	if r.mode != beginWithoutKey {
		mode = "keyed"
	}
	encoded, err := json.Marshal(beginResponse{
		Roles: r.Roles, WorkID: r.WorkID, Idempotency: mode, Resumed: r.mode == beginResumed,
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
	mode := beginWithoutKey
	if work.IdemKey != "" {
		mode = beginWithKey
	}
	if resumed {
		mode = beginResumed
	}
	return BeginResult{mode: mode, WorkID: work.ID, Roles: publicRoles(roster)}
}
