package domain_test

import (
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// reports builds what a gateway says about a model's thinking.
func reports(reasoning, canDisable bool, levels ...string) domain.ModelInfo {
	return domain.ModelInfo{Thinking: domain.Thinking{
		Reported: true, Reasoning: reasoning, CanDisable: canDisable, Levels: levels,
	}}
}

func TestEffortLevelsFollowWhatTheGatewayReports(t *testing.T) {
	t.Parallel()

	full := []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}
	gateway := domain.KindCompatible

	type levelCase struct {
		name string
		kind domain.Kind
		want []string
		info domain.ModelInfo
	}

	cases := []levelCase{
		{name: "a gateway that says nothing gets every level", kind: gateway, want: full},
		{name: "a reasoning model gets every level", kind: gateway, want: full, info: reports(true, true)},
		{name: "a model that does not reason gets none", kind: gateway, want: []string{}, info: reports(false, true)},
		{name: "a model that cannot stop thinking has no none", kind: gateway, want: full[1:], info: reports(true, false)},
		{
			name: "levels the gateway lists win", kind: gateway, want: []string{"low", "high"},
			info: reports(true, true, "low", "high"),
		},
		{name: "Anthropic has no none or minimal", kind: domain.KindAnthropic, want: full[2:]},
		{name: "Gemini stops at high", kind: domain.KindGemini, want: full[:5]},
	}

	for _, c := range cases {
		if got := domain.EffortLevels(c.kind, c.info); !slices.Equal(got, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
