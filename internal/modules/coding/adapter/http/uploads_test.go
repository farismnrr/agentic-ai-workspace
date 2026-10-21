package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type uploadResult struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Media string `json:"media"`
	Size  int64  `json:"size"`
}

func genericUpload(t *testing.T, app *fiber.App, name string, data []byte) (int, uploadResult) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodPost, "/api/projects/p/uploads", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Cookie", "session=test")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var result uploadResult
	_ = json.NewDecoder(res.Body).Decode(&result)
	return res.StatusCode, result
}

func TestGenericUploadAcceptsEverySupportedKind(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	for _, tc := range []struct {
		name, kind string
		head       []byte
	}{
		{"a.png", "image", []byte("\x89PNG\r\n\x1a\n")},
		{"a.jpg", "image", []byte("\xff\xd8\xff")},
		{"a.gif", "image", []byte("GIF89a")},
		{"a.webp", "image", []byte("RIFFxxxxWEBP")},
		{"a.svg", "image", []byte(`<svg>`)},
		{"a.pdf", "pdf", []byte("%PDF-1.7")},
		{"a.txt", "text", []byte("hello")},
		{"a.mp4", "video", []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}},
		{"a.webm", "video", []byte{0x1a, 0x45, 0xdf, 0xa3, 0x88, 0x42, 0x82, 0x84, 'w', 'e', 'b', 'm'}},
		{"a.mov", "video", []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'q', 't', ' ', ' '}},
	} {
		status, result := genericUpload(t, app, tc.name, tc.head)
		if status != fiber.StatusCreated || result.Kind != tc.kind || result.Path == "" {
			t.Errorf("%s: status=%d result=%+v", tc.name, status, result)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, result.Path)); err != nil {
			t.Errorf("%s was not stored: %v", tc.name, err)
		}
	}
}

func TestGenericUploadRejectsBadEmptyOversizedAndUnsafeFiles(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	status, _ := genericUpload(t, app, "fake.pdf", []byte("MZ executable"))
	if status != fiber.StatusUnsupportedMediaType {
		t.Errorf("wrong content status = %d", status)
	}
	if emptyStatus, _ := genericUpload(t, app, "empty.txt", nil); emptyStatus != fiber.StatusBadRequest {
		t.Errorf("empty status = %d", emptyStatus)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("other", "no file")
	_ = form.Close()
	request := httptest.NewRequestWithContext(context.Background(), nethttp.MethodPost, "/api/projects/p/uploads", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("Cookie", "session=test")
	missing, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = missing.Body.Close()
	if missing.StatusCode != fiber.StatusBadRequest {
		t.Errorf("missing file field status = %d", missing.StatusCode)
	}
	tooLarge := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, domain.MaxUploadBytes+1-8)...)
	oversizedStatus, _ := genericUpload(t, app, "large.png", tooLarge)
	if oversizedStatus != fiber.StatusRequestEntityTooLarge {
		t.Errorf("oversized status = %d", oversizedStatus)
	}
	status, result := genericUpload(t, app, "../../x.png", []byte("\x89PNG\r\n\x1a\n"))
	if status != fiber.StatusCreated || result.Path != "uploads/x.png" {
		t.Errorf("unsafe name status=%d result=%+v", status, result)
	}
	if _, err := os.Stat(filepath.Join(root, "x.png")); !os.IsNotExist(err) {
		t.Errorf("unsafe file escaped uploads/: %v", err)
	}
}

func TestGenericUploadCollisionsOwnershipAndLargeImages(t *testing.T) {
	t.Parallel()
	app, _ := newApp(t)
	image := []byte("\x89PNG\r\n\x1a\n")
	first, _ := genericUpload(t, app, "same.png", image)
	second, collided := genericUpload(t, app, "same.png", image)
	if first != fiber.StatusCreated || second != fiber.StatusCreated || collided.Name != "same-2.png" {
		t.Fatalf("collision results: first=%d second=%d %#v", first, second, collided)
	}
	large := append(append([]byte(nil), image...), make([]byte, 12<<20-len(image))...)
	if status, _ := genericUpload(t, app, "large.png", large); status != fiber.StatusCreated {
		t.Fatalf("12 MiB upload status = %d", status)
	}
}

func TestGenericUploadFromAnotherPersonIsNotFound(t *testing.T) {
	t.Parallel()
	status, _ := genericUpload(t, strangerApp(), "a.pdf", []byte("%PDF-1.7"))
	if status != fiber.StatusNotFound {
		t.Fatalf("stranger status = %d", status)
	}
}

func TestFailedLargeUploadLeavesNoPartFilesInTempDir(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	app, _ := newApp(t)
	data := make([]byte, 17<<20)
	data[0] = 0
	status, _ := genericUpload(t, app, "invalid.txt", data)
	if status != fiber.StatusUnsupportedMediaType {
		t.Fatalf("failed upload status = %d, want %d", status, fiber.StatusUnsupportedMediaType)
	}

	err := filepath.WalkDir(tempDir, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".part") {
			t.Errorf("upload temp file remains after failed upload: %s", filePath)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
