package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPassword = "correct horse"

type person struct {
	cookie, token, dir string
}

func TestPeopleNeverShareAWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	sariCookie, clientID := signedInClient(t, app)
	sari := person{cookie: sariCookie, token: connect(t, app, sariCookie, clientID, "coding:read coding:write")}
	sari.dir = userWorkspace(t, app, sari.cookie)

	bobCookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Bob","email":"bob@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	bob := person{cookie: bobCookie, token: connect(t, app, bobCookie, clientID, "coding:read coding:write")}
	bob.dir = userWorkspace(t, app, bob.cookie)

	enableTool(t, app, sari.cookie, "write_file")
	enableTool(t, app, bob.cookie, "write_file")

	callTool(t, app, sari.token, "write_file", `{"path":"secret.txt","content":"SARI ONLY"}`)

	// Bob sees an empty workspace of his own: neither Sari's new file nor her sample files.
	list := callTool(t, app, bob.token, "list_files", `{}`)
	mustNotContain(t, list.Body, "secret.txt", "hello.txt")
	mustContain(t, callTool(t, app, bob.token, "read_file", `{"path":"secret.txt"}`).Body, `"isError":true`)

	// Bob reaching for Sari's folder by name must fail too.
	escape := `{"path":"../` + filepath.Base(sari.dir) + `/secret.txt"}`
	probe := callTool(t, app, bob.token, "read_file", escape)
	mustContain(t, probe.Body, `"isError":true`)
	mustNotContain(t, probe.Body, "SARI ONLY")

	// Bob writing the same path never touches Sari's file.
	callTool(t, app, bob.token, "write_file", `{"path":"secret.txt","content":"BOB"}`)
	mustContain(t, callTool(t, app, sari.token, "read_file", `{"path":"secret.txt"}`).Body, "SARI ONLY")

	if filepath.Dir(sari.dir) != filepath.Dir(bob.dir) || sari.dir == bob.dir {
		t.Fatalf("each person needs a distinct folder, got %q and %q", sari.dir, bob.dir)
	}
}

func TestDeletingAnAccountErasesItsWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read")
	callTool(t, app, token, "list_files", `{}`)

	dir := userWorkspace(t, app, cookie)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the workspace should exist before deletion: %v", err)
	}

	gone := do(t, app, "POST", "/api/auth/delete", `{"password":"`+testPassword+`"}`,
		map[string]string{"Cookie": cookie, "Content-Type": "application/json"})
	if gone.Status != 204 {
		t.Fatalf("account deletion failed: %d %s", gone.Status, gone.Body)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the workspace must be deleted with the account, got %v", err)
	}
}

func mustNotContain(t *testing.T, body []byte, unwanted ...string) {
	t.Helper()

	for _, text := range unwanted {
		if strings.Contains(string(body), text) {
			t.Errorf("response must not contain %q: %s", text, body)
		}
	}
}
