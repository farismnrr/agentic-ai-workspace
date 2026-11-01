package application

import (
	"encoding/json"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// toolRisk distinguishes passive Git/GitHub inspections from operations
// that alter source, issues, CI or releases. It never lowers unknown commands.
func (r *Runner) toolRisk(call Call) domain.Risk {
	risk := r.deps.Tools.Risk(call.Name)
	if call.Name != codingdomain.ToolGit.Name && call.Name != codingdomain.ToolGh.Name {
		return risk
	}
	var in struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(call.Input, &in); err == nil &&
		codingdomain.VCSReadOnly(call.Name, in.Args) {
		return domain.RiskNone
	}
	return risk
}
