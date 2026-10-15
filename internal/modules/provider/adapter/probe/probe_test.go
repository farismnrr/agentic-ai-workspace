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

	want := []domain.ModelInfo{
		{Name: "cx/gpt-6-luna[1m]", ContextWindow: 872000},
		{Name: "claude-x", ContextWindow: 1_000_000},
		{Name: "router/x", ContextWindow: 131072},
		{Name: "plain"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
