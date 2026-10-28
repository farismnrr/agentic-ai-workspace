package stream_test

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"github.com/gofiber/fiber/v3"
)

// serve starts the app on a real port, which streaming needs (app.Test waits for the end of a response).
func serve(t *testing.T, app *fiber.App) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()

	t.Cleanup(func() { _ = app.ShutdownWithTimeout(500 * time.Millisecond) })

	return "http://" + listener.Addr().String()
}

// eventStream reads server-sent events from a URL as lines.
type eventStream struct {
	lines  chan string
	cancel context.CancelFunc
}

func openStream(t *testing.T, url, cookie string) *eventStream {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Cookie", cookie)

	response, err := http.DefaultClient.Do(request) //nolint:bodyclose // closed by the reader goroutine below
	if err != nil {
		t.Fatal(err)
	}

	contentType := response.Header.Get("Content-Type")
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("not an event stream: %d %s", response.StatusCode, contentType)
	}

	stream := &eventStream{lines: make(chan string, 256), cancel: cancel}

	go func() {
		defer func() { _ = response.Body.Close() }()
		defer close(stream.lines)

		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			stream.lines <- scanner.Text()
		}
	}()

	return stream
}

// next waits for a line that contains want and returns it.
func (s *eventStream) next(t *testing.T, want string) string {
	t.Helper()

	deadline := time.After(10 * time.Second)

	for {
		select {
		case line, open := <-s.lines:
			if !open {
				t.Fatalf("the stream ended before %q", want)
			}

			if strings.Contains(line, want) {
				return line
			}
		case <-deadline:
			t.Fatalf("no %q within 10 s", want)
		}
	}
}

func TestATaskStreamPushesTheThreadAndTheModelsTextAsItIsWritten(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.TextThenCall("Working on it now", "f1", "finish",
		map[string]any{"summary": "All done"}))
	a.Gateway.Hold = make(chan struct{})

	base := serve(t, a.Environment.App)
	id := a.Start(t, "Say hello")
	stream := openStream(t, base+"/api/tasks/"+id+"/stream", a.Cookie)

	stream.next(t, `"title":"Say hello"`)
	close(a.Gateway.Hold)

	got := stream.next(t, "event: delta") + stream.next(t, `"text":`)
	if !strings.Contains(got, "delta") {
		t.Fatal("the model's words must arrive as deltas")
	}

	if answer := stream.next(t, `"All done"`); !strings.Contains(answer, `"kind":"message"`) {
		t.Fatalf("an answer arrives as a plain message in the thread: %s", answer)
	}

	stream.next(t, `"status":"done"`)
}

func TestTheWorkspaceStreamAnnouncesChangesAndStaysPrivate(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	base := serve(t, a.Environment.App)
	stream := openStream(t, base+"/api/stream", a.Cookie)

	stream.next(t, "ready")
	id := a.Start(t, "Anything")
	stream.next(t, "event: changed")
	a.WaitFor(t, id, "done")

	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	url := base + "/api/tasks/" + id + "/stream"
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	request.Header.Set("Cookie", other)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("someone else's task stream must be hidden, got %d", response.StatusCode)
	}
}
