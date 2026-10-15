package domain_test

import (
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// claudeStyle builds a model a gateway says thinks like Claude (9router: claude-adaptive or claude-budget).
func claudeStyle(canDisable bool) domain.ModelInfo {
	info := reports(true, canDisable)
	info.Thinking.Format = "claude-adaptive"

	return info
}

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
		{name: "a Claude-style model has no none or minimal", kind: gateway, want: full[2:], info: claudeStyle(true)},
		{name: "a Claude-style model that cannot stop thinking", kind: gateway, want: full[2:], info: claudeStyle(false)},
		{name: "Anthropic has no none or minimal", kind: domain.KindAnthropic, want: full[2:]},
		{name: "Gemini stops at high", kind: domain.KindGemini, want: full[:5]},
	}

	for _, c := range cases {
		if got := domain.EffortLevels(c.kind, c.info); !slices.Equal(got, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOnlyAGatewaysModelsAreGroupedByOwner(t *testing.T) {
	t.Parallel()

	catalog := map[string]domain.ModelInfo{"cx/gpt": {Owner: "cx"}, "gpt-5": {Owner: "system"}}

	gateway := domain.Provider{Kind: domain.KindCompatible, Catalog: catalog}
	if gateway.GroupOf("cx/gpt") != "cx" || gateway.GroupOf("unlisted") != "" {
		t.Fatal("a gateway groups its models by the owner it reports")
	}

	direct := domain.Provider{Kind: domain.KindOpenAI, Catalog: catalog}
	if direct.GroupOf("gpt-5") != "" {
		t.Fatal("a direct provider's models must not be split by its owner label")
	}
}
