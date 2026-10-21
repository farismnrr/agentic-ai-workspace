package http_test

import (
	"context"
	"fmt"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"

	codingfs "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codinghttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type rawResult struct {
	header nethttp.Header
	body   []byte
	status int
}

func requestRaw(t *testing.T, app *fiber.App, rangeValue, ifRange string) rawResult {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet,
		"/api/projects/p/file/raw?path=clip.mp4", nil)
	if rangeValue != "" {
		req.Header.Set("Range", rangeValue)
	}
	if ifRange != "" {
		req.Header.Set("If-Range", ifRange)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	return rawResult{status: res.StatusCode, header: res.Header, body: body}
}

func TestRawSupportsSingleAndSuffixRanges(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	write(t, root, "clip.mp4", "0123456789")
	for _, tc := range []struct{ value, contentRange, body string }{
		{"bytes=2-5", "bytes 2-5/10", "2345"},
		{"bytes=-3", "bytes 7-9/10", "789"},
	} {
		res := requestRaw(t, app, tc.value, "")
		if res.status != nethttp.StatusPartialContent || res.header.Get("Content-Range") != tc.contentRange ||
			string(res.body) != tc.body {
			t.Errorf("Range %q: status=%d range=%q body=%q", tc.value, res.status, res.header.Get("Content-Range"), res.body)
		}
	}
	res := requestRaw(t, app, "bytes=30-40", "")
	if res.status != nethttp.StatusRequestedRangeNotSatisfiable || res.header.Get("Content-Range") != "bytes */10" {
		t.Errorf("unsatisfiable range: status=%d range=%q", res.status, res.header.Get("Content-Range"))
	}
}

func TestRawIgnoresInvalidOrConditionalRanges(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	write(t, root, "clip.mp4", "0123456789")
	for _, value := range []string{"bytes=0-1,4-5", "items=0-1", "bytes=bad"} {
		res := requestRaw(t, app, value, "")
		if res.status != nethttp.StatusOK || string(res.body) != "0123456789" {
			t.Errorf("Range %q: status=%d body=%q", value, res.status, res.body)
		}
	}
	res := requestRaw(t, app, "bytes=0-2", "tag")
	if res.status != nethttp.StatusOK || string(res.body) != "0123456789" {
		t.Errorf("If-Range should send full body: status=%d body=%q", res.status, res.body)
	}
}

func TestRawReturns404ForAnotherOwner(t *testing.T) {
	t.Parallel()
	if status, _ := fetchRaw(t, strangerApp(), "video.mp4"); status != nethttp.StatusNotFound {
		t.Fatalf("other owner's file status = %d", status)
	}
}

type closeProbe struct {
	*os.File
	closed bool
}

func (f *closeProbe) Close() error {
	f.closed = true
	if err := f.File.Close(); err != nil {
		return fmt.Errorf("close probe file: %w", err)
	}
	return nil
}

type observedFiles struct {
	application.FileSystem
	probe *closeProbe
}

func (f observedFiles) Open(rel string) (io.ReadSeekCloser, int64, error) {
	file, size, err := f.FileSystem.Open(rel)
	if err != nil {
		return nil, 0, fmt.Errorf("open observed file: %w", err)
	}
	opened, ok := file.(*os.File)
	if !ok {
		_ = file.Close()
		return nil, 0, domain.ErrDenied
	}
	f.probe.File = opened
	return f.probe, size, nil
}

func TestRawClosesTheFileHandleAfterStreaming(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "clip.mp4", "0123456789")
	base, err := codingfs.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	probe := &closeProbe{}
	workspace := codinghttp.Workspace{Reader: application.NewService(observedFiles{FileSystem: base, probe: probe})}
	pass := func(c fiber.Ctx) error { return c.Next() }
	app := fiber.New()
	codinghttp.NewHandler(owner{workspace: workspace}, quiet{}).Mount(pass, pass, nil)(app)
	_ = requestRaw(t, app, "bytes=1-3", "")
	if !probe.closed {
		t.Fatal("stream did not close its file handle")
	}
}
