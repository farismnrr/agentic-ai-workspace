package application

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type archiveEntry struct {
	name string
	body string
	mode uint32
}

func makeArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			header.SetMode(os.ModeSymlink | 0o777)
		}
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestArchiveUploadSafeAndNeverExtracted(t *testing.T) {
	t.Parallel()
	editor, files := newUploadEditor()
	data := makeArchive(t, archiveEntry{name: "assets/notes.txt", body: "hello zip"})
	stored, err := editor.Upload("example.zip", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind != "archive" || stored.Media != "application/zip" || stored.Path != "uploads/example.zip" {
		t.Fatalf("unexpected stored archive: %#v", stored)
	}
	if !bytes.Equal(files.files[stored.Path], data) {
		t.Fatal("ZIP bytes not preserved")
	}
	if _, found := files.files["assets/notes.txt"]; found {
		t.Fatal("upload must not unpack ZIP")
	}
}

func TestArchiveUploadRejectsTraversalBombLinksAndBadContent(t *testing.T) {
	t.Parallel()
	cases := map[string][]byte{
		"traversal":        makeArchive(t, archiveEntry{name: "../escape.txt", body: "oops"}),
		"hidden_traversal": makeArchive(t, archiveEntry{name: "foo/../../escape.txt", body: "oops"}),
		"internal_dotdot":  makeArchive(t, archiveEntry{name: "foo/../safe.txt", body: "oops"}),
		"absolute":         makeArchive(t, archiveEntry{name: "/etc/passwd", body: "oops"}),
		"windows":          makeArchive(t, archiveEntry{name: "..\\escape.txt", body: "oops"}),
		"nested":           makeArchive(t, archiveEntry{name: "inner.zip", body: "oops"}),
		"symlink":          makeArchive(t, archiveEntry{name: "link", mode: 1, body: "/etc/passwd"}),
		"bomb":             makeArchive(t, archiveEntry{name: "huge.txt", body: strings.Repeat("a", 5<<20)}),
		"invalid":          []byte("PK\x03\x04not-a-real-zip"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			editor, files := newUploadEditor()
			_, err := editor.Upload(name+".zip", int64(len(data)), bytes.NewReader(data))
			if !errors.Is(err, errUnsafeArchive) {
				t.Fatalf("expected unsafe archive, got %v", err)
			}
			for key := range files.files {
				if strings.HasSuffix(key, ".zip") || strings.HasSuffix(key, ".part") {
					t.Errorf("unsafe archive left on disk: %s", key)
				}
			}
		})
	}
}

func TestArchiveUploadRejectsRenamedExecutable(t *testing.T) {
	t.Parallel()
	editor, _ := newUploadEditor()
	_, err := editor.Upload("malware.zip", 9, strings.NewReader("MZ runtime"))
	if !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("renamed executable accepted: %v", err)
	}
}
