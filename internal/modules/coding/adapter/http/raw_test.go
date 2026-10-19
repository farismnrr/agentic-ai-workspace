package http_test

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// fetchRaw answers with the status and headers of the file, and closes the body.
func fetchRaw(t *testing.T, app *fiber.App, target string) (int, nethttp.Header) {
	t.Helper()

	url := "/api/projects/p/file/raw?path=" + target
	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, url, nil)

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	return res.StatusCode, res.Header
}

func write(t *testing.T, root, name, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAFileIsServedAsItIsAndNeverAsAPage(t *testing.T) {
	t.Parallel()

	app, root := newApp(t)
	write(t, root, "documents/deck.pptx", "PK-pptx")
	write(t, root, "documents/a.pdf", "%PDF-1.7")
	write(t, root, "documents/x.html", "<script>1</script>")

	const pptx = "application/vnd.openxmlformats-officedocument.presentationml.presentation"

	cases := []struct{ target, media, disposition string }{
		{"documents/deck.pptx", pptx, "attachment"},
		{"documents/a.pdf", "application/pdf", "inline"},
		{"documents/a.pdf&download=1", "application/pdf", "attachment"},
		{"documents/x.html", "application/octet-stream", "attachment"},
	}

	for _, tc := range cases {
		_, header := fetchRaw(t, app, tc.target)
		disposition := header.Get("Content-Disposition")

		if header.Get("Content-Type") != tc.media || !strings.HasPrefix(disposition, tc.disposition) {
			t.Errorf("%s: %q", tc.target, header)
		}

		if header.Get("X-Content-Type-Options") != "nosniff" || header.Get("Content-Security-Policy") == "" {
			t.Errorf("%s: missing protective headers: %q", tc.target, header)
		}
	}
}

func TestAMissingOrSecretFileIsNotFound(t *testing.T) {
	t.Parallel()

	app, root := newApp(t)
	write(t, root, ".env", "KEY=1")

	for _, target := range []string{"nope.pdf", ".env", "../etc/passwd"} {
		if status, _ := fetchRaw(t, app, target); status != nethttp.StatusNotFound {
			t.Errorf("%s: status %d, want 404", target, status)
		}
	}
}
