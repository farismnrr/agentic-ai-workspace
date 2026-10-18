// Package searxng searches the web through a SearXNG server the person runs. The address comes from the server's own
// settings, never from the model, so it may be on the local network.
package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

const (
	timeout   = 20 * time.Second
	maxBody   = 1 << 20
	maxSnip   = 300
	userAgent = "agentic-ai-workspace/1 (web search for the agent)"
)

// Client talks to one SearXNG server.
type Client struct {
	http *http.Client
	base string
}

// New builds a client for a base address such as http://searxng:8080.
func New(baseURL string) *Client {
	return &Client{base: strings.TrimRight(strings.TrimSpace(baseURL), "/"), http: &http.Client{Timeout: timeout}}
}

type response struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

// Search returns up to limit hits for the query. SearXNG answers JSON only when its settings allow the format;
// a 403 usually means they do not.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]domain.Result, error) {
	address := c.base + "/search?" + url.Values{"q": {query}, "format": {"json"}, "safesearch": {"1"}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return nil, domain.ErrUnavailable
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, domain.ErrUnavailable
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %w", resp.Status, application.StatusOf(resp.StatusCode))
	}

	var body response
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: unreadable answer", domain.ErrUnavailable)
	}

	results := make([]domain.Result, 0, limit)

	for _, hit := range body.Results {
		if len(results) == limit {
			break
		}

		if _, err := domain.ValidURL(hit.URL); err != nil {
			continue
		}

		results = append(results, domain.Result{
			Title: clip(hit.Title, maxSnip), URL: hit.URL, Snippet: clip(hit.Content, maxSnip),
		})
	}

	return results, nil
}

func clip(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit]) + "…"
	}

	return text
}
