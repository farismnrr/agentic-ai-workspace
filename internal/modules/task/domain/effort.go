package domain

import "slices"

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

// order lists the levels from the least to the most thinking.
var order = []Effort{
	EffortNone, EffortMinimal, EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax, EffortUltra,
}

// Nearest is the level a model takes that is closest to the one asked for: the same, else the highest below it,
// else the lowest above it. It is empty when the model takes none or the level is not one we know.
func Nearest(levels []string, want Effort) Effort {
	at := slices.Index(order, want)
	if at < 0 {
		return EffortDefault
	}

	for i := at; i >= 0; i-- {
		if slices.Contains(levels, string(order[i])) {
			return order[i]
		}
	}

	for _, level := range order[at+1:] {
		if slices.Contains(levels, string(level)) {
			return level
		}
	}

	return EffortDefault
}

// Valid reports whether e is empty or a known level.
func (e Effort) Valid() bool {
	switch e {
	case EffortDefault, EffortNone, EffortMinimal, EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax,
		EffortUltra:
		return true
	}

	return false
}
