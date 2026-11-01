package projects_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func uploadTestZIP(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	header := &zip.FileHeader{Name: "src/index.js", Method: zip.Store}
	part, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("console.log('hello');")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestResumableZIPHTTPBlackBoxAuthorizesResumesAndPublishes(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	var project string
	err := env.DB.QueryRow("SELECT id FROM projects WHERE source = 'ChatGPT' ORDER BY created_at DESC LIMIT 1").Scan(&project)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project + "/uploads/resumable"
	raw := uploadTestZIP(t)
	meta := base64.StdEncoding.EncodeToString([]byte("sample.zip"))
	headers := map[string]string{"Cookie": cookie, "Tus-Resumable": "1.0.0",
		"Upload-Length": strconv.Itoa(len(raw)), "Upload-Metadata": "filename " + meta}
	denied := apptest.Do(t, env.App, "POST", base, "", map[string]string{
		"Tus-Resumable": "1.0.0", "Upload-Length": strconv.Itoa(len(raw)), "Upload-Metadata": "filename " + meta})
	if denied.Status != 401 {
		t.Fatalf("unauthed create: %d %s", denied.Status, denied.Body)
	}
	alien := map[string]string{}
	for k, v := range headers {
		alien[k] = v
	}
	alien["Origin"] = "https://hostile.example"
	csrf := apptest.Do(t, env.App, "POST", base, "", alien)
	if csrf.Status != 403 {
		t.Fatalf("cross-origin create: %d %s", csrf.Status, csrf.Body)
	}
	created := apptest.Do(t, env.App, "POST", base, "", headers)
	if created.Status != 201 {
		t.Fatalf("create: %d %s", created.Status, created.Body)
	}
	url := created.Header.Get("Location")
	if !strings.HasPrefix(url, base+"/") {
		t.Fatalf("missing location: %q", url)
	}
	head := apptest.Do(t, env.App, "HEAD", url, "", map[string]string{"Cookie": cookie})
	if head.Status != 200 || head.Header.Get("Upload-Offset") != "0" {
		t.Fatalf("initial HEAD: %d %+v", head.Status, head.Header)
	}
	half := len(raw) / 2
	patchHeaders := map[string]string{"Cookie": cookie, "Tus-Resumable": "1.0.0",
		"Content-Type": "application/offset+octet-stream", "Upload-Offset": "0"}
	part := apptest.Do(t, env.App, "PATCH", url, string(raw[:half]), patchHeaders)
	if part.Status != 204 || part.Header.Get("Upload-Offset") != fmt.Sprint(half) {
		t.Fatalf("first PATCH: %d %s", part.Status, part.Body)
	}
	replay := apptest.Do(t, env.App, "PATCH", url, string(raw[:half]), patchHeaders)
	if replay.Status != 409 || replay.Header.Get("Upload-Offset") != fmt.Sprint(half) {
		t.Fatalf("replay must not duplicate bytes: %d %+v", replay.Status, replay.Header)
	}
	patchHeaders["Upload-Offset"] = fmt.Sprint(half)
	finish := apptest.Do(t, env.App, "PATCH", url, string(raw[half:]), patchHeaders)
	if finish.Status != 204 || finish.Header.Get("Upload-Offset") != strconv.Itoa(len(raw)) {
		t.Fatalf("tus PATCH must acknowledge final offset with 204: %d %s", finish.Status, finish.Body)
	}
	var uploadPath string
	var again apptest.Reply
	for attempts := 0; attempts < 100; attempts++ {
		again = apptest.Do(t, env.App, "HEAD", url, "", map[string]string{"Cookie": cookie})
		if again.Status == 200 && again.Header.Get("Upload-State") == "ready" {
			uploadPath = again.Header.Get("Upload-Path")
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.HasPrefix(uploadPath, "uploads/") {
		t.Fatalf("asynchronous publish missing path: %+v", again)
	}
	other := apptest.Do(t, env.App, "HEAD", url, "", nil)
	if other.Status != 401 {
		t.Fatalf("private upload status leaked: %+v", other)
	}
}
