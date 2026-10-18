package openai_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
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

	if _, err := openai.New(server.URL+"/v1", "k", local).Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	return strings.ReplaceAll(seen, " ", "")
}

func TestAScreenshotFollowsTheToolMessageAsAUserMessage(t *testing.T) {
	t.Parallel()

	text := sentWithScreenshot(t)
	tool := strings.Index(text, `"role":"tool"`)
	user := strings.Index(text, `"role":"user"`)

	if tool < 0 || user < tool || !strings.Contains(text, `"image_url":{"url":"data:image/png;base64,UE5H"}`) {
		t.Fatalf("the picture is a user message after the tool message: %s", text)
	}
}
