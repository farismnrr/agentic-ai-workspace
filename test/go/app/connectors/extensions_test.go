package connectors_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

const goodManifest = `{"manifest_version":3,"name":"Tab Notes","version":"1.0.0","description":"Notes for tabs",
 "permissions":["storage","tabs"],"host_permissions":["https://example.com/*"],
 "action":{"default_popup":"popup.html"},"icons":{"16":"icon16.png"}}`

func writeBuild(t *testing.T, a apptest.AgentRig, manifest string) {
	t.Helper()

	for name, content := range map[string]string{
		"dist/manifest.json": manifest, "dist/popup.html": "<h1>hi</h1>", "dist/icon16.png": "png",
	} {
		path := a.File(name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func deployScript(path string) []string {
	return []string{
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "DevOps: package the extension", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "devops", "brief": "Package the extension"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "devops", "brief": "Package the extension"}),
		apptest.ToolCall("x1", "deploy_extension", map[string]string{"path": path}),
		apptest.ToolCall("r1", "report", map[string]string{"summary": "Packaged"}),
		apptest.FinishCall("Your extension is ready"),
	}
}

func TestADeployedExtensionIsListedAndDownloadableAsAZip(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, deployScript("dist")...)
	apptest.EnableTool(t, a.Environment.App, a.Cookie, "run_command")
	writeBuild(t, a, goodManifest)

	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Package it","team":true}`, apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "done")

	listed := apptest.Do(t, a.Environment.App, "GET", "/api/extensions", "", apptest.JSONCookie(a.Cookie))
	apptest.MustContain(t, listed.Body, `"name":"Tab Notes"`, `"version":"1.0.0"`,
		"See the address and title of your open tabs",
		"Read and change what you see on example.com", `"downloadUrl":"/api/extensions/`+a.Project+`/package"`)

	zipped := apptest.Do(t, a.Environment.App, "GET", "/api/extensions/"+a.Project+"/package", "",
		apptest.JSONCookie(a.Cookie))
	if zipped.Status != 200 {
		t.Fatalf("download failed: %d", zipped.Status)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipped.Body), int64(len(zipped.Body)))
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)
	}

	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "manifest.json") || !strings.Contains(joined, "popup.html") {
		t.Fatalf("the package holds the build with the manifest at its root: %v", names)
	}

	thread := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	apptest.MustContain(t, []byte(thread), `"kind":"extension"`, `/app/extensions/`+a.Project)

	if gone := apptest.Do(t, a.Environment.App, "DELETE", "/api/extensions/"+a.Project, "",
		apptest.JSONCookie(a.Cookie)); gone.Status != 204 {
		t.Fatalf("removing failed: %d", gone.Status)
	}
}

func TestAnExtensionWithAMissingFileIsNotPackagedAndSaysWhatToFix(t *testing.T) {
	t.Parallel()

	broken := strings.Replace(goodManifest, `"popup.html"`, `"missing-popup.html"`, 1)
	a := apptest.NewAgentRig(t, deployScript("dist")...)
	apptest.EnableTool(t, a.Environment.App, a.Cookie, "run_command")
	writeBuild(t, a, broken)

	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Package it","team":true}`, apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.WaitApproving(t, id, "done")

	apptest.MustContain(t, apptest.Do(t, a.Environment.App, "GET", "/api/extensions", "",
		apptest.JSONCookie(a.Cookie)).Body, `"extensions":[]`)

	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	if !strings.Contains(events, "missing-popup.html") {
		t.Fatalf("the builder is told which file is missing: %s", events)
	}
}
