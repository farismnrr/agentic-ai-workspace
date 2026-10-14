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

	if want := []string{"gemini-3.8-flash", "gemini-3.5-flash-lite"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
