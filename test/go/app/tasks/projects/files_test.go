package projects_test

import (
	"net/url"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestThePersonSeesTheProjectFilesButNotTheSecrets(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	a.PutFile(t, "index.html", "<h1>hi</h1>")
	a.PutFile(t, "src/app.js", "x")
	a.PutFile(t, ".env", "SECRET=1")

	list := a.Files(t, "GET", "/files", "")
	entries, _ := list.JSON(t)["entries"].([]any)

	if list.Status != 200 || len(entries) != 2 {
		t.Fatalf("the tree shows the folder and the file, not .env: %d %s", list.Status, list.Body)
	}

	if first, _ := entries[0].(map[string]any); first["name"] != "src" || first["dir"] != true {
		t.Fatalf("folders come first: %v", entries)
	}

	file := a.Files(t, "GET", "/file?path="+url.QueryEscape("index.html"), "")
	if body := file.JSON(t); file.Status != 200 || body["content"] != "<h1>hi</h1>" || body["hash"] == "" {
		t.Fatalf("a file comes with its hash: %d %s", file.Status, file.Body)
	}

	if hidden := a.Files(t, "GET", "/file?path=.env", ""); hidden.Status != 404 {
		t.Fatalf("a secret file does not exist for the page: %d", hidden.Status)
	}

	missing := apptest.Do(t, a.Environment.App, "GET", "/api/projects/nope0000/files", "", apptest.JSONCookie(a.Cookie))
	if missing.Status != 404 {
		t.Fatalf("a project that is not theirs is not found: %d", missing.Status)
	}
}

func TestAnEditIsRefusedWhenTheAgentChangedTheFileMeanwhile(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	a.PutFile(t, "page.html", "one")

	hash, _ := a.Files(t, "GET", "/file?path=page.html", "").JSON(t)["hash"].(string)

	saved := a.Files(t, "PUT", "/file", `{"path":"page.html","content":"two","baseHash":"`+hash+`"}`)
	if saved.Status != 200 || a.Read("page.html") != "two" {
		t.Fatalf("an edit of the file as opened is saved: %d %s", saved.Status, saved.Body)
	}

	// The agent changes the file after the person opened it: the old hash no longer fits.
	a.PutFile(t, "page.html", "agent wrote this")

	stale := a.Files(t, "PUT", "/file", `{"path":"page.html","content":"three","baseHash":"`+hash+`"}`)
	if stale.Status != 409 || a.Read("page.html") != "agent wrote this" {
		t.Fatalf("a stale edit must not overwrite the agent's work: %d %s", stale.Status, stale.Body)
	}

	if fresh := a.Files(t, "PUT", "/file", `{"path":"new/page.html","content":"x","baseHash":""}`); fresh.Status != 200 {
		t.Fatalf("a new file needs no hash: %d", fresh.Status)
	}

	if clash := a.Files(t, "PUT", "/file", `{"path":"new/page.html","content":"y","baseHash":""}`); clash.Status != 409 {
		t.Fatalf("a new file must not replace an existing one: %d", clash.Status)
	}
}

func TestAFolderCanBeDeletedWithEverythingInItButOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	a.PutFile(t, "assets/img/a.png", "binary-ish")
	a.PutFile(t, "assets/b.txt", "text")
	a.PutFile(t, "keep.txt", "stays")

	if plain := a.Files(t, "DELETE", "/file?path=assets", ""); plain.Status != 400 {
		t.Fatalf("a folder is never deleted by the plain file call: %d", plain.Status)
	}

	if wrong := a.Files(t, "DELETE", "/file?path=keep.txt&folder=1", ""); wrong.Status != 400 || a.Read("keep.txt") == "" {
		t.Fatalf("a file is not a folder: %d", wrong.Status)
	}

	gone := a.Files(t, "DELETE", "/file?path=assets&folder=1", "")
	removed := a.Read("assets/b.txt") == "" && a.Read("assets/img/a.png") == ""
	if gone.Status != 204 || !removed || a.Read("keep.txt") != "stays" {
		t.Fatalf("the folder goes with its files and nothing else: %d", gone.Status)
	}

	if root := a.Files(t, "DELETE", "/file?path=&folder=1", ""); root.Status != 400 {
		t.Fatalf("the project folder itself cannot be deleted: %d", root.Status)
	}
}
