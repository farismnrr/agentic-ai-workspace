package anthropic_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/anthropic"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// sentWithScreenshot sends one result that carries a screenshot and returns the JSON the server received.
func sentWithScreenshot(t *testing.T) string {
	t.Helper()

	seen := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = string(raw)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	request := application.Request{Messages: []domain.Message{{
		Role: domain.RoleUser,
		Blocks: []domain.Block{
			{Kind: domain.BlockToolResult, ID: "t1", Text: "Screenshot of /", Data: []byte("PNG")},
			{Kind: domain.BlockToolResult, ID: "t2", Text: "plain"},
		},
	}}}

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	return strings.ReplaceAll(seen, " ", "")
}

func TestAScreenshotTravelsInsideTheToolResultBeforeItsText(t *testing.T) {
	t.Parallel()

	text := sentWithScreenshot(t)
	image := strings.Index(text, `"type":"image"`)
	words := strings.Index(text, `"text":"Screenshotof/"`)

	if image < 0 || words < image || !strings.Contains(text, `"media_type":"image/png","data":"UE5H"`) {
		t.Fatalf("the picture comes first in the result: %s", text)
	}

	if !strings.Contains(text, `"content":"plain"`) {
		t.Fatalf("a result without a picture stays a plain string: %s", text)
	}
}
