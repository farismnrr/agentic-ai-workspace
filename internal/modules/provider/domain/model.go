package domain

import "strings"

// Thinking is what a provider said about how a model thinks. The zero value means it said nothing.
type Thinking struct {
	// Format is how the gateway says the model thinks (9router: openai, claude-adaptive, ...); empty when unsaid.
	Format string
	// Levels are the effort levels the provider lists for the model itself, when it does.
	Levels []string
	// Reported is true when the provider described the model's thinking at all.
	Reported bool
	// Reasoning is whether the model reasons; only meaningful when Reported.
	Reasoning bool
	// CanDisable is false when the model cannot switch thinking off; only meaningful when Reported.
	CanDisable bool
}

// ModelInfo is a model a provider listed, with what it reported about it.
type ModelInfo struct {
	Name string
	// Owner is who serves the model, as a gateway reports it (9router: cc, cx, kr, a combo name); empty when unsaid.
	Owner    string
	Thinking Thinking
	// ContextWindow is zero when the provider did not say.
	ContextWindow int64
}

// claudeFormat starts the thinking format a gateway reports for Claude-style models (claude-adaptive, claude-budget).
const claudeFormat = "claude"

// The effort levels, from the least to the most thinking.
var (
	levelsAnthropic = []string{"low", "medium", "high", "xhigh", "max"}
	levelsOpenAI    = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}
	levelsGemini    = []string{"none", "minimal", "low", "medium", "high"}
)

// EffortLevels are the effort levels worth offering for a model. A gateway that describes its models decides:
// a model that does not reason gets none, and "none" is left out when it cannot switch thinking off. A gateway
// that says nothing, or lists something we know nothing about, gets every level, because the gateway translates
// them for the provider behind it and the provider names any it cannot take. Direct providers get the levels
// their own documentation lists.
func EffortLevels(kind Kind, model ModelInfo) []string {
	thinking := model.Thinking
	if thinking.Reported && !thinking.Reasoning {
		return []string{}
	}

	levels := append([]string{}, kindLevels(kind)...)

	switch {
	case len(thinking.Levels) > 0:
		levels = append([]string{}, thinking.Levels...)
	case strings.HasPrefix(thinking.Format, claudeFormat):
		// A Claude-style model, whoever serves it, takes Anthropic's levels: there is no "minimal" or "none".
		levels = append([]string{}, levelsAnthropic...)
	}

	if thinking.Reported && !thinking.CanDisable {
		levels = without(levels, "none")
	}

	return levels
}

func kindLevels(kind Kind) []string {
	switch kind {
	case KindAnthropic:
		return levelsAnthropic
	case KindGemini:
		return levelsGemini
	case KindOpenAI, KindCompatible:
		return levelsOpenAI
	}

	return levelsOpenAI
}

func without(levels []string, drop string) []string {
	kept := make([]string, 0, len(levels))

	for _, level := range levels {
		if level != drop {
			kept = append(kept, level)
		}
	}

	return kept
}

// GroupOf is who serves a model, for providers that host many: a gateway's models are grouped by the owner it
// reports. A direct provider serves its own models, so they are not split (OpenAI calls everything "system").
func (p Provider) GroupOf(model string) string {
	if !p.Kind.CustomAddress() {
		return ""
	}

	return p.Catalog[model].Owner
}
