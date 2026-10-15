package domain

// Effort is how hard the model should think about a task. Empty means the provider's own default.
type Effort string

// Effort levels, in the words the providers use: OpenAI-style gateways call it reasoning_effort and
// Anthropic output_config.effort. Not every model accepts every level; the provider says so when it does not.
const (
	EffortDefault Effort = ""
	EffortLow     Effort = "low"
	EffortMedium  Effort = "medium"
	EffortHigh    Effort = "high"
	EffortXHigh   Effort = "xhigh"
	EffortMax     Effort = "max"
)

// Valid reports whether e is empty or a known level.
func (e Effort) Valid() bool {
	switch e {
	case EffortDefault, EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax:
		return true
	}

	return false
}
