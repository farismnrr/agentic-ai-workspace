package openverse_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/openverse"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

func serve(t *testing.T, handle func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(server.Close)

	return server
}

func TestSearchReadsLicencesAndAsksForCommercialUse(t *testing.T) {
	t.Parallel()

	var asked string

	sample, err := os.ReadFile("testdata/sample.json")
	if err != nil {
		t.Fatal(err)
	}

	server := serve(t, func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RawQuery
		_, _ = w.Write(sample)
	})

	images, err := openverse.New(server.URL, server.Client()).Search(context.Background(), "coffee shop", 5)
	if err != nil || len(images) != 2 {
		t.Fatalf("images = %+v, %v", images, err)
	}

	first := images[0]
	if first.License != "CC BY-SA 2.0" || first.Creator != "pixeljones" || images[1].License != "CC0" {
		t.Fatalf("licences are written the way people read them: %+v", images)
	}

	if first.Source != "openverse/flickr" {
		t.Fatalf("the original source is named: %+v", first)
	}

	if !strings.Contains(asked, "license=cc0%2Cpdm") || !strings.Contains(asked, "mature=false") {
		t.Fatalf("only pictures that are CC0 or public domain, no mature ones: %q", asked)
	}
}

func TestARateLimitIsSaidPlainly(t *testing.T) {
	t.Parallel()

	server := serve(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })

	_, err := openverse.New(server.URL, server.Client()).Search(context.Background(), "x y", 3)
	if !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("429 is busy: %v", err)
	}
}
