package application

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type uploadFiles struct{ writer *uploadWriter }

func (f uploadFiles) Stat(rel string) (domain.Entry, error) {
	data, ok := f.writer.files[rel]
	if !ok {
		return domain.Entry{}, domain.ErrNotFound
	}
	return domain.Entry{Path: rel, Size: int64(len(data))}, nil
}
func (uploadFiles) ReadDir(string) ([]domain.Entry, error)       { return nil, nil }
func (uploadFiles) ReadFile(string, int64) ([]byte, bool, error) { return nil, false, nil }
func (f uploadFiles) Open(rel string) (io.ReadSeekCloser, int64, error) {
	data, ok := f.writer.files[rel]
	if !ok {
		return nil, 0, domain.ErrNotFound
	}
	return uploadReadSeeker{Reader: bytes.NewReader(data)}, int64(len(data)), nil
}
func (uploadFiles) Walk(string, func(domain.Entry) error) error { return nil }

type uploadReadSeeker struct{ *bytes.Reader }

func (uploadReadSeeker) Close() error { return nil }

type uploadWriter struct {
	files       map[string][]byte
	beforeWrite func(string)
}

func (w *uploadWriter) Create(rel string, src io.Reader) error {
	if w.beforeWrite != nil {
		w.beforeWrite(rel)
	}
	if _, exists := w.files[rel]; exists {
		return domain.ErrExists
	}
	data, err := io.ReadAll(src)
	if err == nil {
		w.files[rel] = data
	} else {
		err = fmt.Errorf("read upload fixture: %w", err)
	}
	return err
}
func (w *uploadWriter) WriteFile(rel string, data []byte) error { w.files[rel] = data; return nil }
func (w *uploadWriter) Remove(rel string) error                 { delete(w.files, rel); return nil }
func (w *uploadWriter) RemoveAll(string) error                  { return nil }
func (w *uploadWriter) Rename(from, to string) error {
	if _, exists := w.files[to]; exists {
		return domain.ErrExists
	}
	w.files[to] = w.files[from]
	delete(w.files, from)
	return nil
}

type uploadIDs struct{ n int }

func (i *uploadIDs) ID() (string, error) {
	i.n++
	return "upload-" + strconv.Itoa(i.n), nil
}

func newUploadEditor() (*Editor, *uploadWriter) {
	writer := &uploadWriter{files: make(map[string][]byte)}
	return NewEditor(uploadFiles{writer}, writer, nil, &uploadIDs{}), writer
}

func TestUploadStoresSafelyAndAvoidsCollisions(t *testing.T) {
	t.Parallel()
	editor, writer := newUploadEditor()
	writer.files[path.Join("uploads", "report.pdf")] = []byte("existing")
	content := []byte("%PDF-1.7 sample")
	stored, err := editor.Upload("../../Report.pdf", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Path != "uploads/report-2.pdf" || stored.Name != "report-2.pdf" || stored.Size != int64(len(content)) {
		t.Fatalf("stored = %#v", stored)
	}
	if !bytes.Equal(writer.files[stored.Path], content) {
		t.Fatal("stored content differs")
	}
}

func TestUploadDoesNotOverwriteFileCreatedAfterNameCheck(t *testing.T) {
	t.Parallel()
	editor, writer := newUploadEditor()
	writer.beforeWrite = func(rel string) {
		if rel == "uploads/report.pdf" {
			writer.files[rel] = []byte("agent file")
		}
	}
	content := []byte("%PDF-1.7 upload")
	stored, err := editor.Upload("report.pdf", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Path != "uploads/report-2.pdf" {
		t.Fatalf("stored path = %q", stored.Path)
	}
	if string(writer.files["uploads/report.pdf"]) != "agent file" {
		t.Fatal("file created between name check and write was overwritten")
	}
}

func TestUploadRejectsDeclaredOversizeAndEmpty(t *testing.T) {
	t.Parallel()
	editor, _ := newUploadEditor()
	oversized := int64(domain.MaxUploadBytes + 1)
	image := bytes.NewReader([]byte("\x89PNG\r\n\x1a\n"))
	if _, err := editor.Upload("image.png", oversized, image); !errors.Is(err, domain.ErrTooLarge) {
		t.Fatalf("oversize error = %v", err)
	}
	if _, err := editor.Upload("empty.txt", 0, bytes.NewReader(nil)); !errors.Is(err, domain.ErrEmpty) {
		t.Fatalf("empty error = %v", err)
	}
}

func TestUploadRemovesTemporaryFileAfterContentFailure(t *testing.T) {
	t.Parallel()
	editor, writer := newUploadEditor()
	content := append(bytes.Repeat([]byte("a"), 1024), 0xff, 0xfe)
	size := int64(len(content))
	if _, err := editor.Upload("notes.txt", size, bytes.NewReader(content)); !errors.Is(err, domain.ErrBinary) {
		t.Fatalf("invalid text error = %v", err)
	}
	for name := range writer.files {
		if path.Ext(name) == ".part" {
			t.Fatalf("temporary upload remained: %s", name)
		}
	}
}
