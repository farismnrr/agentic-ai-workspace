package commons_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/commons"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

// serve answers every request with the saved Commons reply.
func serve(t *testing.T) *httptest.Server {
	t.Helper()

	sample, err := os.ReadFile("testdata/sample.json")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(sample)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestSearchKeepsPhotosInOrderWithPlainTextCredits(t *testing.T) {
	t.Parallel()

	server := serve(t)

	images, err := commons.New(server.URL, server.Client()).Search(context.Background(), "coffee shop", 8)
	if err != nil || len(images) != 2 {
		t.Fatalf("pictures only, not trademarked ones or videos: %+v, %v", images, err)
	}

	checkOrder(t, images)
	checkCredit(t, images[1])
}

func checkOrder(t *testing.T, images []domain.Image) {
	t.Helper()

	first, second := images[0], images[1]
	if first.Title != "First" || first.License != "Public domain" || second.Title != "Second one" {
		t.Fatalf("in the order Commons ranked them: %+v", images)
	}
}

func checkCredit(t *testing.T, image domain.Image) {
	t.Helper()

	if image.Creator != "Jane & Co" || image.Width != 1600 {
		t.Fatalf("markup is removed from the credit: %+v", image)
	}

	if image.URL != "https://upload.wikimedia.org/thumb/second.jpg" {
		t.Fatalf("the smaller copy is used: %+v", image)
	}
}
