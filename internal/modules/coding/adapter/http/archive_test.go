package http_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func zipped(t *testing.T, name, data string) []byte {
	t.Helper()
	var buf bytes.Buffer
	out := zip.NewWriter(&buf)
	file, err := out.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestZIPChatUploadRemainsAnArchive(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	data := zipped(t, "reports/summary.md", "hello")
	status, result := genericUpload(t, app, "bundle.zip", data)
	if status != fiber.StatusCreated || result.Kind != "archive" ||
		result.Path != "uploads/bundle.zip" {
		t.Fatalf("ZIP upload status=%d result=%+v", status, result)
	}
	safeRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = safeRoot.Close() }()
	stored, err := safeRoot.ReadFile("uploads/bundle.zip")
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("stored ZIP bytes incorrect: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "reports", "summary.md")); !os.IsNotExist(err) {
		t.Fatal("upload extracted archive onto server")
	}
}

func TestZIPChatUploadRejectsTraversalAndDamagedArchive(t *testing.T) {
	t.Parallel()
	app, root := newApp(t)
	for _, data := range [][]byte{
		zipped(t, "../escape.txt", "attack"),
		[]byte("PK\x03\x04broken"),
	} {
		status, _ := genericUpload(t, app, "archive.zip", data)
		if status != fiber.StatusUnsupportedMediaType {
			t.Errorf("unsafe ZIP status=%d, want 415", status)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "uploads", "archive.zip")); !os.IsNotExist(err) {
		t.Error("unsafe ZIP persisted")
	}
}
