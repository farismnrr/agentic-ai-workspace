package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestEveryResponseNamesTheBuildVersion(t *testing.T) {
	frontend := fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}
	app := New(frontend, Settings{Version: "abc1234"})

	for _, path := range []string{"/", "/api/health"} {
		res, err := app.Test(httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		if got := res.Header.Get("X-App-Version"); got != "abc1234" {
			t.Errorf("%s: X-App-Version %q, want abc1234", path, got)
		}

		_ = res.Body.Close()
	}

	res, err := app.Test(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/health", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	var body healthPayload
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body.Status != "ok" || body.Version != "abc1234" {
		t.Errorf("health = %+v, want ok and abc1234", body)
	}
}

func TestVersionDefaultsToDev(t *testing.T) {
	app := New(fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, Settings{})

	res, err := app.Test(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/health", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	if got := res.Header.Get("X-App-Version"); got != "dev" {
		t.Errorf("X-App-Version %q, want dev", got)
	}
}
