package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	callTimeout = 5 * time.Minute
	maxResponse = 8 << 20
	maxMessage  = 300
)

var errRedirect = errors.New("the model address redirects elsewhere")

// ErrEventStream means the provider answered with server-sent events although a single JSON answer was asked for.
// Some gateways do this for every request to certain models (9router does for Claude), so the caller can read
// the answer as a stream instead.
var ErrEventStream = errors.New("the model answered with an event stream")

// Client posts JSON to a model provider and maps failures to domain.ModelError.
type Client struct {
	http *http.Client
}

// New builds a Client that may only reach what the policy allows. Redirects are
// refused so a key is never sent to another host.
func New(policy egress.Policy) *Client {
	return &Client{http: &http.Client{
		Timeout:       callTimeout,
		Transport:     egress.Transport(policy),
		CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect },
	}}
}

// Post sends body as JSON with the given headers and decodes a 2xx answer into out.
func (c *Client) Post(ctx context.Context, url string, headers map[string]string, body, out any) error {
	response, err := c.open(ctx, url, headers, body)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return &domain.ModelError{Message: "the model's answer was cut off", Retryable: true}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return statusError(response.StatusCode, data)
	}

	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("data:")) {
		return ErrEventStream
	}

	if err := json.Unmarshal(data, out); err != nil {
		return &domain.ModelError{Message: "the model sent an answer this app cannot read"}
	}

	return nil
}

const (
	maxEventLine = 4 << 20
	scanBuffer   = 64 << 10
)

// Stream posts body and feeds each server-sent event of a 2xx answer to handle.
// Returning an error from handle stops the stream.
func (c *Client) Stream(
	ctx context.Context, url string, headers map[string]string, body any, handle func(event, data string) error,
) error {
	response, err := c.open(ctx, url, headers, body)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(response.Body, maxResponse))

		return statusError(response.StatusCode, data)
	}

	return readEvents(ctx, response.Body, handle)
}

// readEvents parses a server-sent event stream and calls handle for every complete event.
func readEvents(ctx context.Context, body io.Reader, handle func(event, data string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, scanBuffer), maxEventLine)

	var current pendingEvent

	for scanner.Scan() {
		ready, err := current.feed(scanner.Text(), handle)
		if err != nil {
			return err
		}

		if ready {
			current = pendingEvent{}
		}
	}

	if ctx.Err() != nil {
		return fmt.Errorf("model call cancelled: %w", ctx.Err())
	}

	if scanner.Err() != nil {
		return &domain.ModelError{Message: "the model's answer was cut off", Retryable: true}
	}

	return nil
}

// pendingEvent is the event being read: a name and its data lines, ended by a blank line.
type pendingEvent struct {
	name string
	data []string
}

// feed takes one line; ready reports that a blank line completed an event.
func (p *pendingEvent) feed(line string, handle func(event, data string) error) (bool, error) {
	switch {
	case line == "":
		if len(p.data) == 0 {
			return true, nil
		}

		return true, handle(p.name, strings.Join(p.data, "\n"))
	case strings.HasPrefix(line, "event:"):
		p.name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
	case strings.HasPrefix(line, "data:"):
		p.data = append(p.data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
	}

	return false, nil
}

func (c *Client) open(ctx context.Context, url string, headers map[string]string, body any) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode model request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build model request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("model call cancelled: %w", ctx.Err())
		}

		return nil, &domain.ModelError{Message: "could not reach the model: " + shorten(err.Error()), Retryable: true}
	}

	return response, nil
}
