package account_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// modelServer pretends to be an OpenAI-compatible gateway that accepts one key.
func modelServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" || r.Header.Get("Authorization") != "Bearer sk-good-key-1234" {
			http.Error(w, "no", http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"},{"id":"model-b"}]}`))
	}))
	t.Cleanup(server.Close)

	return server
}

func addProvider(t *testing.T, app apptest.Environment, cookie, baseURL, key string) apptest.Reply {
	t.Helper()

	body := `{"kind":"compatible","name":"Local test gateway","baseUrl":"` + baseURL + `","apiKey":"` + key + `"}`

	return apptest.Do(t, app.App, "POST", "/api/providers", body, apptest.JSONCookie(cookie))
}

func TestProviderKeyIsSealedVerifiedAndNeverReturned(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	server := modelServer(t)

	created := addProvider(t, env, cookie, server.URL, "sk-good-key-1234")
	if created.Status != 201 || strings.Contains(string(created.Body), "sk-good-key-1234") {
		t.Fatalf("created without leaking the key: %d %s", created.Status, created.Body)
	}

	id, _ := created.JSON(t)["id"].(string)
	verified := apptest.Do(t, env.App, "POST", "/api/providers/"+id+"/verify", "", apptest.JSONCookie(cookie))
	apptest.MustContain(t, verified.Body, `"verified":true`, "model-a", `"keyHint":"sk-…1234"`)

	listed := apptest.Do(t, env.App, "GET", "/api/providers", "", apptest.JSONCookie(cookie))
	if strings.Contains(string(listed.Body), "sk-good-key-1234") {
		t.Fatalf("listing must not expose keys: %s", listed.Body)
	}

	assertNotOnDisk(t, env.DataDir, "sk-good-key-1234")

	apptest.MustContain(t, apptest.Do(t, env.App, "GET", "/api/models", "", apptest.JSONCookie(cookie)).Body,
		`"model":"model-b"`)
}

func TestWrongKeyFailsVerificationAndDisabledProvidersOfferNoModels(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	server := modelServer(t)

	bad, _ := addProvider(t, env, cookie, server.URL, "sk-wrong").JSON(t)["id"].(string)
	if got := apptest.Do(t, env.App, "POST", "/api/providers/"+bad+"/verify", "",
		apptest.JSONCookie(cookie)).Status; got != 422 {
		t.Fatalf("a rejected key must be a 422, got %d", got)
	}

	good, _ := addProvider(t, env, cookie, server.URL, "sk-good-key-1234").JSON(t)["id"].(string)
	apptest.Do(t, env.App, "POST", "/api/providers/"+good+"/verify", "", apptest.JSONCookie(cookie))
	apptest.Do(t, env.App, "PUT", "/api/providers/"+good, `{"enabled":false}`, apptest.JSONCookie(cookie))

	models := apptest.Do(t, env.App, "GET", "/api/models", "", apptest.JSONCookie(cookie))
	if strings.Contains(string(models.Body), "model-a") {
		t.Fatalf("a disabled provider must offer no models: %s", models.Body)
	}
}

func TestProviderInputIsValidatedAndOwned(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)

	for _, body := range []string{
		`{"kind":"nope","name":"x","apiKey":"k"}`,
		`{"kind":"compatible","name":"x","baseUrl":"javascript:alert(1)","apiKey":"k"}`,
		`{"kind":"compatible","name":"x","baseUrl":"","apiKey":"k"}`,
		`{"kind":"anthropic","name":"","apiKey":"k"}`,
		`{"kind":"anthropic","name":"x","apiKey":" "}`,
	} {
		if got := apptest.Do(t, env.App, "POST", "/api/providers", body, apptest.JSONCookie(cookie)).Status; got != 400 {
			t.Errorf("%s must be rejected, got %d", body, got)
		}
	}

	id, _ := addProvider(t, env, cookie, modelServer(t).URL, "sk-good-key-1234").JSON(t)["id"].(string)
	other := apptest.CookieOf(t, apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	if got := apptest.Do(t, env.App, "DELETE", "/api/providers/"+id, "", apptest.JSONCookie(other)).Status; got != 404 {
		t.Fatalf("someone else's provider must not be reachable, got %d", got)
	}

	if got := apptest.Do(t, env.App, "GET", "/api/providers", "", nil).Status; got != 401 {
		t.Fatalf("anonymous requests must be refused, got %d", got)
	}
}

func TestPreferencesRoundTripAndGetIDs(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)

	empty := apptest.Do(t, env.App, "GET", "/api/preferences", "", apptest.JSONCookie(cookie))
	apptest.MustContain(t, empty.Body, `"rules":[]`)

	saved := apptest.Do(t, env.App, "PUT", "/api/preferences",
		`{"personaName":"Sari","about":"I run a bakery",`+
			`"rules":[{"text":" Never change prices ","on":true},{"text":"","on":true}]}`,
		apptest.JSONCookie(cookie))

	var prefs struct {
		Rules []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(saved.Body, &prefs); err != nil || len(prefs.Rules) != 1 || prefs.Rules[0].ID == "" {
		t.Fatalf("blank rules are dropped and new ones get ids: %s", saved.Body)
	}

	if prefs.Rules[0].Text != "Never change prices" {
		t.Errorf("rule text must be trimmed, got %q", prefs.Rules[0].Text)
	}

	long := `{"about":"` + strings.Repeat("x", 1001) + `","rules":[]}`
	if got := apptest.Do(t, env.App, "PUT", "/api/preferences", long, apptest.JSONCookie(cookie)).Status; got != 400 {
		t.Fatalf("oversized text must be rejected, got %d", got)
	}
}

// assertNotOnDisk fails if secret appears in the database or its write-ahead log.
func assertNotOnDisk(t *testing.T, dataDir, secret string) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dataDir, "workspace.db*"))
	if err != nil || len(files) == 0 {
		t.Fatalf("expected database files in %s: %v", dataDir, err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file) //nolint:gosec // reads the test's own database
		if err != nil {
			t.Fatal(err)
		}

		if strings.Contains(string(data), secret) {
			t.Fatalf("the secret is stored in plain text in %s", filepath.Base(file))
		}
	}
}
