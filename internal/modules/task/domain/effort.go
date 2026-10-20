package domain

// Effort is how hard the model should think about a task. Empty means the provider's own default.
type Effort string

// Effort levels, in the words the providers use: OpenAI-style gateways call it reasoning_effort and
// Anthropic output_config.effort. None and minimal are OpenAI's and Gemini's, not Anthropic's. Not every
// model accepts every level; the provider says so when it does not.
const (
	EffortDefault Effort = ""
	EffortNone    Effort = "none"
	EffortMinimal Effort = "minimal"
	EffortLow     Effort = "low"
	EffortMedium  Effort = "medium"
	EffortHigh    Effort = "high"
	EffortXHigh   Effort = "xhigh"
	EffortMax     Effort = "max"
	EffortUltra   Effort = "ultra"
)

// Valid reports whether e is empty or a known level.
func (e Effort) Valid() bool {
	switch e {
	case EffortDefault, EffortNone, EffortMinimal, EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax,
		EffortUltra:
		return true
	}

	return false
}
