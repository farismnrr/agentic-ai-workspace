package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func registerClient(t *testing.T, app *fiber.App, name string) string {
	t.Helper()

	dcr := do(t, app, "POST", "/oauth/register",
		`{"client_name":"`+name+`","redirect_uris":["`+redirect+`"],"token_endpoint_auth_method":"none"}`, jsonHeaders())
	if dcr.Status != 201 {
		t.Fatalf("DCR failed: %d %s", dcr.Status, dcr.Body)
	}

	id, _ := dcr.JSON(t)["client_id"].(string)

	return id
}

type projectRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

func projectsOf(t *testing.T, app *fiber.App, cookie string) []projectRow {
	t.Helper()

	var rows []projectRow
	if err := json.Unmarshal(do(t, app, "GET", "/api/projects", "", jsonCookie(cookie)).Body, &rows); err != nil {
		t.Fatal(err)
	}

	return rows
}

func TestEachConnectedAppWorksInAProjectNamedAfterIt(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, chatgpt := signedInClient(t, env.app)
	gemini := registerClient(t, env.app, "Gemini")
	enableTool(t, env.app, cookie, "write_file")

	for _, clientID := range []string{chatgpt, gemini, chatgpt} {
		token := connect(t, env.app, cookie, clientID, "coding:read coding:write")
		callTool(t, env.app, token, "write_file", `{"path":"note.txt","content":"from `+clientID+`"}`)
	}

	byName := map[string]projectRow{}

	for _, row := range projectsOf(t, env.app, cookie) {
		if _, twice := byName[row.Name]; twice {
			t.Fatalf("reconnecting must reuse the app's project, got two called %q", row.Name)
		}

		byName[row.Name] = row
	}

	for _, app := range []string{"ChatGPT", "Gemini"} {
		row, ok := byName[app]
		if !ok || row.Source != app {
			t.Fatalf("%s should have a project of its own, got %+v", app, byName)
		}

		data, err := os.ReadFile(filepath.Join(env.dataDir, "projects", row.ID, "note.txt"))
		if err != nil || !strings.HasPrefix(string(data), "from ") {
			t.Fatalf("%s's files belong in its project: %v %q", app, err, data)
		}
	}

	if string(mustRead(t, env.dataDir, byName["ChatGPT"].ID)) == string(mustRead(t, env.dataDir, byName["Gemini"].ID)) {
		t.Fatal("two apps must never share a folder")
	}

	rows := do(t, env.app, "GET", "/api/activity", "", jsonCookie(cookie)).Body
	mustContain(t, rows, `"client":"ChatGPT"`, `"client":"Gemini"`)
}

func mustRead(t *testing.T, dataDir, project string) []byte {
	t.Helper()

	path := filepath.Join(dataDir, "projects", project, "note.txt")

	data, err := os.ReadFile(path) //nolint:gosec // the test's own temp folder
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestAppsFirstCallsAtOnceStillMakeOneProject(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Rae","email":"rae@example.com","password":"correct horse"}`, jsonHeaders()))
	token := connect(t, app, cookie, registerClient(t, app, "Racer"), "coding:read")

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			callTool(t, app, token, "list_files", `{}`)
		}()
	}

	wg.Wait()

	count := 0

	for _, row := range projectsOf(t, app, cookie) {
		if row.Source == "Racer" {
			count++
		}
	}

	if count != 1 {
		t.Fatalf("one app must have exactly one project, got %d", count)
	}
}
