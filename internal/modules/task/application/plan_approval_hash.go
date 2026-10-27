package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func planDecisionHash(decision domain.PlanDecisionKind, revision int64, note string) (string, error) {
	encoded, err := json.Marshal(struct {
		Decision domain.PlanDecisionKind
		Note     string
		Revision int64
	}{Decision: decision, Note: note, Revision: revision})
	if err != nil {
		return "", fmt.Errorf("encode plan decision hash input: %w", err)
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}
