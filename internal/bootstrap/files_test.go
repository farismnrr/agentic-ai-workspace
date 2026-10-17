package bootstrap

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func (a agentRig) put(t *testing.T, name, content string) {
	t.Helper()

	path := a.file(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (a agentRig) files(t *testing.T, method, route, body string) reply {
	t.Helper()

	return do(t, a.env.app, method, "/api/projects/"+a.project+route, body, jsonCookie(a.cookie))
}

func TestThePersonSeesTheProjectFilesButNotTheSecrets(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)
	a.put(t, "index.html", "<h1>hi</h1>")
	a.put(t, "src/app.js", "x")
	a.put(t, ".env", "SECRET=1")

	list := a.files(t, "GET", "/files", "")
	entries, _ := list.JSON(t)["entries"].([]any)

	if list.Status != 200 || len(entries) != 2 {
		t.Fatalf("the tree shows the folder and the file, not .env: %d %s", list.Status, list.Body)
	}

	if first, _ := entries[0].(map[string]any); first["name"] != "src" || first["dir"] != true {
		t.Fatalf("folders come first: %v", entries)
	}

	file := a.files(t, "GET", "/file?path="+url.QueryEscape("index.html"), "")
	if body := file.JSON(t); file.Status != 200 || body["content"] != "<h1>hi</h1>" || body["hash"] == "" {
		t.Fatalf("a file comes with its hash: %d %s", file.Status, file.Body)
	}

	if hidden := a.files(t, "GET", "/file?path=.env", ""); hidden.Status != 404 {
		t.Fatalf("a secret file does not exist for the page: %d", hidden.Status)
	}

	missing := do(t, a.env.app, "GET", "/api/projects/nope0000/files", "", jsonCookie(a.cookie))
	if missing.Status != 404 {
		t.Fatalf("a project that is not theirs is not found: %d", missing.Status)
	}
}

func TestAnEditIsRefusedWhenTheAgentChangedTheFileMeanwhile(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)
	a.put(t, "page.html", "one")

	hash, _ := a.files(t, "GET", "/file?path=page.html", "").JSON(t)["hash"].(string)

	saved := a.files(t, "PUT", "/file", `{"path":"page.html","content":"two","baseHash":"`+hash+`"}`)
	if saved.Status != 200 || a.read("page.html") != "two" {
		t.Fatalf("an edit of the file as opened is saved: %d %s", saved.Status, saved.Body)
	}

	// The agent changes the file after the person opened it: the old hash no longer fits.
	a.put(t, "page.html", "agent wrote this")

	stale := a.files(t, "PUT", "/file", `{"path":"page.html","content":"three","baseHash":"`+hash+`"}`)
	if stale.Status != 409 || a.read("page.html") != "agent wrote this" {
		t.Fatalf("a stale edit must not overwrite the agent's work: %d %s", stale.Status, stale.Body)
	}

	if fresh := a.files(t, "PUT", "/file", `{"path":"new/page.html","content":"x","baseHash":""}`); fresh.Status != 200 {
		t.Fatalf("a new file needs no hash: %d", fresh.Status)
	}

	if clash := a.files(t, "PUT", "/file", `{"path":"new/page.html","content":"y","baseHash":""}`); clash.Status != 409 {
		t.Fatalf("a new file must not replace an existing one: %d", clash.Status)
	}
}

func TestAFileCanBeDeletedAndTheOpenPagesAreToldWhenFilesChange(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)
	a.put(t, "old.txt", "bye")

	if gone := a.files(t, "DELETE", "/file?path=old.txt", ""); gone.Status != 204 || a.read("old.txt") != "" {
		t.Fatalf("delete: %d", gone.Status)
	}

	hub := taskapp.NewHub()
	signals, stop := hubChanges{hub: hub}.Follow("p1")

	defer stop()

	hub.FilesChanged("other")
	hub.FilesChanged("p1")

	select {
	case <-signals:
	case <-time.After(2 * time.Second):
		t.Fatal("a page following the project is told when its files change")
	}
}
