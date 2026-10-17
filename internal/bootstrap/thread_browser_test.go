package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/webassets"
)

// threadMessage is what the agent says before it starts the page: the kind of markdown a model writes.
const threadMessage = "Ini isi proyek sekarang.\n\n**Folder di paling atas:**\n" +
	"- `erp-tahu/`: aplikasi ERP untuk usaha tahu.\n" +
	"  - `static/`: tampilan web.\n- `pengeluaran/`: folder kedua.\n\n" +
	"| File | Isi |\n|---|---|\n| `app.py` | **program** utama |\n\n" +
	"```sh\nmake run\n```\nLihat https://example.com untuk contoh."

// threadPage has a few things the screenshot report should find: pale text, a broken image and a wide box.
const threadPage = `<!doctype html><title>Warung</title><body style="margin:0;font-family:Georgia,serif">
<header style="padding:48px;background:#3b2a20;color:#f5e9dc"><h1>Kopi enak setiap pagi</h1></header>
<p style="color:#bbb;padding:0 48px">Teks pucat yang sulit dibaca</p><img src="x.png">
<div style="width:1500px;height:30px;background:#ccc"></div></body>`

// saying turns a scripted tool call into one where the model also says something.
func saying(text, call string) string {
	var body map[string]any

	_ = json.Unmarshal([]byte(call), &body)
	choices, _ := body["choices"].([]any)
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	message["content"] = text
	out, _ := json.Marshal(body)

	return string(out)
}

// TestServeThreadForABrowser runs a task whose model starts a page, takes two screenshots of it with the real
// sandbox and browser and runs a check, then waits for an external browser run (Playwright) to look at the
// thread. It does nothing unless THREAD_BROWSER_DIR is set. It writes info.json there and stops when a file
// named done appears.
func TestServeThreadForABrowser(t *testing.T) {
	dir, image := os.Getenv("THREAD_BROWSER_DIR"), os.Getenv("SANDBOX_TEST_IMAGE")
	if dir == "" || image == "" {
		t.Skip("set THREAD_BROWSER_DIR and SANDBOX_TEST_IMAGE to serve a thread for a browser run")
	}

	frontend, err := webassets.Assets()
	if err != nil {
		t.Fatal(err)
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	base := "http://" + ln.Addr().String()
	env := newEnvServing(t, func(c *config.Config) {
		c.SandboxImage, c.PublicURL, c.CookieSecure = image, base, false
		c.AllowedOrigins, c.CORSOrigins = []string{base}, []string{base}
	}, frontend)

	command := "echo " + base64.StdEncoding.EncodeToString([]byte(threadPage)) +
		" | base64 -d > index.html && exec python3 -m http.server $PORT --bind 127.0.0.1"
	a := rigOn(t, env,
		saying(threadMessage, toolCall("s1", "start_site", map[string]any{"command": command, "port": 8126})),
		saying("Saya lihat tampilannya di layar komputer.", toolCall("p1", "screenshot", map[string]any{})),
		saying("Sekarang di layar HP.", toolCall("p2", "screenshot", map[string]any{"device": "mobile"})),
		saying("Saya jalankan satu pemeriksaan.", toolCall("c1", "run_command", map[string]any{"command": "echo checked"})),
		finishCall("Halaman jalan. Ada **teks pucat** dan gambar rusak."),
	)

	go func() { _ = env.app.Listener(ln) }()

	t.Cleanup(func() { _ = env.app.Shutdown() })

	id := a.start(t, "Tunjukkan halamannya")
	a.waitApproving(t, id, "review")

	info, _ := json.Marshal(map[string]string{
		"base": base, "path": "/app/projects/" + a.project + "/tasks/" + id,
		"email": "sari@example.com", "password": testPassword,
	})
	if err = os.WriteFile(filepath.Join(dir, "info.json"), info, 0o600); err != nil { //nolint:gosec // test-only folder
		t.Fatal(err)
	}

	for deadline := time.Now().Add(5 * time.Minute); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		if _, statErr := os.Stat(filepath.Join(dir, "done")); statErr == nil { //nolint:gosec // test-only folder
			return
		}
	}
}
