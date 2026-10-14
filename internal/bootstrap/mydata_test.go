package bootstrap

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipText(t *testing.T, data []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the export must be a zip: %v", err)
	}

	files := map[string]string{}

	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}

		body, err := io.ReadAll(handle)
		_ = handle.Close()

		if err != nil {
			t.Fatal(err)
		}

		files[file.Name] = string(body)
	}

	return files
}

func TestExportHoldsEverythingButSecrets(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	id, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Bakery site"}`, jsonCookie(cookie)).JSON(t)["id"].(string)

	menu := filepath.Join(env.dataDir, "projects", id, "menu.txt")
	if err := os.WriteFile(menu, []byte("croissant"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := do(t, env.app, "GET", "/api/account/export", "", nil).Status; got != 401 {
		t.Fatalf("export needs a session, got %d", got)
	}

	res := do(t, env.app, "GET", "/api/account/export", "", jsonCookie(cookie))
	if res.Status != 200 || res.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("export failed: %d %v", res.Status, res.Header)
	}

	files := zipText(t, res.Body)
	account := files["account.json"]
	if !strings.Contains(account, "sari@example.com") || strings.Contains(account, "password") {
		t.Fatalf("account.json should hold the profile and no password: %s", account)
	}

	if got := files["projects/Bakery-site-"+id+"/files/menu.txt"]; got != "croissant" {
		t.Fatalf("project files must be included, got %v", files)
	}
}

func TestDeletingAllTasksKeepsProjects(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	id, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Keep"}`, jsonCookie(cookie)).JSON(t)["id"].(string)

	if got := do(t, env.app, "DELETE", "/api/tasks", "", jsonCookie(cookie)).Status; got != 204 {
		t.Fatalf("delete all tasks failed: %d", got)
	}

	mustContain(t, do(t, env.app, "GET", "/api/projects", "", jsonCookie(cookie)).Body, id)
}

func TestDeletingTheAccountRemovesItsFoldersAndNeedsThePassword(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	id, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Gone"}`, jsonCookie(cookie)).JSON(t)["id"].(string)
	folder := filepath.Join(env.dataDir, "projects", id)

	deleteWith := func(password string) int {
		body := `{"password":"` + password + `"}`

		return do(t, env.app, "POST", "/api/auth/delete", body, jsonCookie(cookie)).Status
	}

	if got := deleteWith("wrong wrong"); got != 400 {
		t.Fatalf("a wrong password must be refused, got %d", got)
	}

	if _, err := os.Stat(folder); err != nil {
		t.Fatalf("nothing may be removed before the password is right: %v", err)
	}

	if got := deleteWith("correct horse"); got != 204 {
		t.Fatalf("delete failed: %d", got)
	}

	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Fatalf("the project folder must be gone: %v", err)
	}

	if got := passwordLogin(t, env.app, "correct horse"); got != 401 {
		t.Fatalf("the account must be gone, login gave %d", got)
	}
}
