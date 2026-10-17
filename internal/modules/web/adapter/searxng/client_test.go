package searxng_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/searxng"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

func TestSearchReturnsReadableHitsAndDropsOddAddresses(t *testing.T) {
	t.Parallel()

	var asked string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"results":[
			{"title":"Kopi  Teras","url":"https://example.com/kopi","content":"Warung   kopi\nBandung"},
			{"title":"Bad","url":"javascript:alert(1)","content":"x"},
			{"title":"Two","url":"https://example.org/2","content":"y"},
			{"title":"Three","url":"https://example.org/3","content":"z"}]}`))
	}))
	t.Cleanup(server.Close)

	results, err := searxng.New(server.URL+"/").Search(context.Background(), "warung kopi", 2)
	if err != nil || len(results) != 2 || results[0].Title != "Kopi Teras" || results[0].Snippet != "Warung kopi Bandung" {
		t.Fatalf("results = %+v, %v", results, err)
	}

	if asked != "format=json&q=warung+kopi&safesearch=1" {
		t.Fatalf("the query asks for JSON and safe results: %q", asked)
	}
}

func TestSearchReportsWhyItFailed(t *testing.T) {
	t.Parallel()

	statuses := map[int]error{http.StatusForbidden: domain.ErrUnavailable, http.StatusTooManyRequests: domain.ErrBusy}
	for status, want := range statuses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))

		_, err := searxng.New(server.URL).Search(context.Background(), "kopi", 3)
		server.Close()

		if !errors.Is(err, want) {
			t.Errorf("status %d: %v, want %v", status, err, want)
		}
	}

	_, err := searxng.New("http://127.0.0.1:1").Search(context.Background(), "kopi", 3)
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("a server that is down is unavailable: %v", err)
	}
}
