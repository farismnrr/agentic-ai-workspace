package projects_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// Kept as a real HTTP-level multipart-free client. Data in the ZIP is stored,
// not deflated, so this actually exercises a 50+ MiB upload over 7 PATCHes.
type zipZeroReader struct{}

func (zipZeroReader) Read(b []byte) (int, error) {
	clear(b)
	return len(b), nil
}

func TestResumableZIPHTTP50MiBChunked(t *testing.T) {
	var payload bytes.Buffer
	z := zip.NewWriter(&payload)
	w, err := z.CreateHeader(&zip.FileHeader{Name: "assets/data.bin", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(w, zipZeroReader{}, 50<<20); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	raw := payload.Bytes()
	// The storage design reserves file size + 128MiB free headroom. Skip
	// on a genuinely full CI runner rather than weakening the production guard.
	var disk syscall.Statfs_t
	if err := syscall.Statfs(t.TempDir(), &disk); err != nil {
		t.Fatal(err)
	}
	if int64(disk.Bavail)*int64(disk.Bsize) < int64(len(raw))+(160<<20) {
		t.Skip("test runner has insufficient free disk for a 50MiB ZIP plus staging headroom")
	}

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	var project string
	if err := env.DB.QueryRow("SELECT id FROM projects WHERE source = 'ChatGPT' ORDER BY created_at DESC LIMIT 1").Scan(&project); err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project + "/uploads/resumable"
	created := apptest.Do(t, env.App, "POST", base, "", map[string]string{
		"Cookie": cookie, "Tus-Resumable": "1.0.0",
		"Upload-Length":   strconv.Itoa(len(raw)),
		"Upload-Metadata": "filename " + base64.StdEncoding.EncodeToString([]byte("large-50mb.zip")),
	})
	if created.Status != 201 {
		t.Fatalf("create: %d %s", created.Status, created.Body)
	}
	location := created.Header.Get("Location")
	var offset int
	for offset < len(raw) {
		end := offset + 8<<20
		if end > len(raw) {
			end = len(raw)
		}
		resp := apptest.Do(t, env.App, "PATCH", location, string(raw[offset:end]), map[string]string{
			"Cookie": cookie, "Tus-Resumable": "1.0.0",
			"Content-Type":  "application/offset+octet-stream",
			"Upload-Offset": strconv.Itoa(offset),
		})
		if resp.Status != 204 || resp.Header.Get("Upload-Offset") != strconv.Itoa(end) {
			t.Fatalf("PATCH at %d: status %d, offset %s, body=%s", offset, resp.Status, resp.Header.Get("Upload-Offset"), resp.Body)
		}
		offset = end
	}
	var ready bool
	for tries := 0; tries < 300; tries++ {
		response := apptest.Do(t, env.App, "HEAD", location, "", map[string]string{"Cookie": cookie})
		if response.Status == 200 && response.Header.Get("Upload-State") == "ready" {
			if response.Header.Get("Upload-Offset") != strconv.Itoa(len(raw)) {
				t.Fatalf("incorrect final offset: %v", response.Header)
			}
			if response.Header.Get("Upload-Path") == "" {
				t.Fatalf("missing final path: %v", response.Header)
			}
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatal("50MiB ZIP did not become ready after validation")
	}
}
