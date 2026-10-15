package probe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/probe"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// local lets the probe reach the loopback test server.
var local = egress.Policy{Rule: egress.LocalOnly}

const catalogue = `{"data":[{"id":"models/gemini-3.8-flash"},{"id":"models/gemini-embedding-001"},` +
	`{"id":"models/gemini-3.6-flash-preview-tts"},{"id":"models/gemini-3-pro-image-preview"},` +
	`{"id":"models/imagen-4"},{"id":"models/gemini-3.5-flash-lite"}]}`

func TestGeminiListsOnlyChatModelsWithoutThePrefix(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" || r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(catalogue))
	}))
	t.Cleanup(server.Close)

	got, err := probe.New(local, local).Models(context.Background(),
		domain.Credential{Kind: domain.KindGemini, BaseURL: server.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(got))
	for _, model := range got {
		names = append(names, model.Name)
	}

	if want := []string{"gemini-3.8-flash", "gemini-3.5-flash-lite"}; !slices.Equal(names, want) {
		t.Fatalf("got %v, want %v", names, want)
	}
}

// A gateway such as 9router lists each model's window under capabilities; Anthropic and OpenRouter use their own
// names. All are read, and a model that says nothing keeps a zero window.
func TestTheContextWindowIsReadFromWhicheverFieldTheProviderUses(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` +
			`{"id":"cx/gpt-6-luna[1m]","capabilities":{"contextWindow":872000},"context_length":872000},` +
			`{"id":"claude-x","max_input_tokens":1000000},` +
			`{"id":"router/x","context_length":131072},` +
			`{"id":"plain"}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := probe.New(local, local).Models(context.Background(),
		domain.Credential{Kind: domain.KindCompatible, BaseURL: server.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int64{"cx/gpt-6-luna[1m]": 872000, "claude-x": 1_000_000, "router/x": 131072, "plain": 0}
	if len(got) != len(want) {
		t.Fatalf("got %d models, want %d", len(got), len(want))
	}

	for _, model := range got {
		if model.ContextWindow != want[model.Name] {
			t.Fatalf("%s: window %d, want %d", model.Name, model.ContextWindow, want[model.Name])
		}
	}
}

// sameThinking compares what two reports say about a model's thinking.
func sameThinking(a, b domain.Thinking) bool {
	return slices.Equal(a.Levels, b.Levels) && a.Reported == b.Reported && a.Reasoning == b.Reasoning &&
		a.CanDisable == b.CanDisable && a.Format == b.Format
}

func TestWhatAGatewayReportsAboutThinkingIsKept(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` +
			`{"id":"cx/gpt","capabilities":{"reasoning":true,"thinkingCanDisable":true}},` +
			`{"id":"cc/claude","capabilities":{"reasoning":true,"thinkingCanDisable":false}},` +
			`{"id":"cx/review","capabilities":{"reasoning":false}},` +
			`{"id":"listed","capabilities":{"reasoningEffort":["low","high"]}},` +
			`{"id":"fmt","capabilities":{"reasoning":true,"thinkingFormat":"claude-adaptive"}},` +
			`{"id":"silent"}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := probe.New(local, local).Models(context.Background(),
		domain.Credential{Kind: domain.KindCompatible, BaseURL: server.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.Thinking{
		{Reported: true, Reasoning: true, CanDisable: true},
		{Reported: true, Reasoning: true},
		{Reported: true, CanDisable: true},
		{Reported: true, Reasoning: true, CanDisable: true, Levels: []string{"low", "high"}},
		{Reported: true, Reasoning: true, CanDisable: true, Format: "claude-adaptive"},
		{},
	}

	for i, model := range got {
		if !sameThinking(model.Thinking, want[i]) {
			t.Fatalf("%s: got %+v, want %+v", model.Name, model.Thinking, want[i])
		}
	}
}

// 9router reports who serves each model as owned_by. Without it the part of the id before a slash is used, and a
// plain id (a combo, or a hosted provider's model) has no owner.
func TestTheOwnerOfAModelComesFromOwnedByThenTheIDPrefix(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"cc/claude","owned_by":"cc"},{"id":"kr/glm-5"},` +
			`{"id":"my-combo"},{"id":"gpt-5","owned_by":"system"}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := probe.New(local, local).Models(context.Background(),
		domain.Credential{Kind: domain.KindCompatible, BaseURL: server.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	owners := map[string]string{}
	for _, model := range got {
		owners[model.Name] = model.Owner
	}

	want := map[string]string{"cc/claude": "cc", "kr/glm-5": "kr", "my-combo": "", "gpt-5": "system"}
	for name, owner := range want {
		if owners[name] != owner {
			t.Fatalf("%s: owner %q, want %q", name, owners[name], owner)
		}
	}
}
