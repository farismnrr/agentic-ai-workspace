package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	callTimeout = 5 * time.Minute
	maxResponse = 8 << 20
	maxMessage  = 300
)

var errRedirect = errors.New("the model address redirects elsewhere")

// Client posts JSON to a model provider and maps failures to domain.ModelError.
type Client struct {
	http *http.Client
}

// New builds a Client. Redirects are refused so a key is never sent to another host.
func New() *Client {
	return &Client{http: &http.Client{
		Timeout:       callTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect },
	}}
}

// Post sends body as JSON with the given headers and decodes a 2xx answer into out.
func (c *Client) Post(ctx context.Context, url string, headers map[string]string, body, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode model request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build model request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("model call cancelled: %w", ctx.Err())
		}

		return &domain.ModelError{Message: "could not reach the model: " + shorten(err.Error()), Retryable: true}
	}

	defer func() { _ = response.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return &domain.ModelError{Message: "the model's answer was cut off", Retryable: true}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return statusError(response.StatusCode, data)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return &domain.ModelError{Message: "the model sent an answer this app cannot read"}
	}

	return nil
}

// statusError turns an HTTP failure into a ModelError; rate limits and outages may be retried.
func statusError(status int, data []byte) error {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	_ = json.Unmarshal(data, &body)

	message := body.Error.Message
	if message == "" {
		message = fmt.Sprintf("the provider answered HTTP %d", status)
	}

	retryable := status == http.StatusTooManyRequests || status >= http.StatusInternalServerError

	return &domain.ModelError{Message: shorten(message), Retryable: retryable}
}

func shorten(text string) string {
	runes := []rune(text)
	if len(runes) <= maxMessage {
		return text
	}

	return string(runes[:maxMessage]) + "…"
}
