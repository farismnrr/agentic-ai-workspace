package bootstrap

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodManifest = `{"manifest_version":3,"name":"Tab Notes","version":"1.0.0","description":"Notes for tabs",
 "permissions":["storage","tabs"],"host_permissions":["https://example.com/*"],
 "action":{"default_popup":"popup.html"},"icons":{"16":"icon16.png"}}`

func writeBuild(t *testing.T, a agentRig, manifest string) {
	t.Helper()

	for name, content := range map[string]string{
		"dist/manifest.json": manifest, "dist/popup.html": "<h1>hi</h1>", "dist/icon16.png": "png",
	} {
		path := a.file(name)
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
		toolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "DevOps: package the extension", "state": "pending"},
		}}),
		toolCall("d0", "delegate", map[string]string{"role": "devops", "brief": "Package the extension"}),
		toolCall("d1", "delegate", map[string]string{"role": "devops", "brief": "Package the extension"}),
		toolCall("x1", "deploy_extension", map[string]string{"path": path}),
		toolCall("r1", "report", map[string]string{"summary": "Packaged"}),
		finishCall("Your extension is ready"),
	}
}

func TestADeployedExtensionIsListedAndDownloadableAsAZip(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, deployScript("dist")...)
	enableTool(t, a.env.app, a.cookie, "run_command")
	writeBuild(t, a, goodManifest)

	created := do(t, a.env.app, "POST", "/api/tasks",
		`{"projectId":"`+a.project+`","prompt":"Package it","team":true}`, jsonCookie(a.cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.waitApproving(t, id, "done")

	listed := do(t, a.env.app, "GET", "/api/extensions", "", jsonCookie(a.cookie))
	mustContain(t, listed.Body, `"name":"Tab Notes"`, `"version":"1.0.0"`, "See the address and title of your open tabs",
		"Read and change what you see on example.com", `"downloadUrl":"/api/extensions/`+a.project+`/package"`)

	zipped := do(t, a.env.app, "GET", "/api/extensions/"+a.project+"/package", "", jsonCookie(a.cookie))
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

	thread := string(do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(a.cookie)).Body)
	mustContain(t, []byte(thread), `"kind":"extension"`, `/app/extensions/`+a.project)

	if gone := do(t, a.env.app, "DELETE", "/api/extensions/"+a.project, "", jsonCookie(a.cookie)); gone.Status != 204 {
		t.Fatalf("removing failed: %d", gone.Status)
	}
}

func TestAnExtensionWithAMissingFileIsNotPackagedAndSaysWhatToFix(t *testing.T) {
	t.Parallel()

	broken := strings.Replace(goodManifest, `"popup.html"`, `"missing-popup.html"`, 1)
	a := newAgentRig(t, deployScript("dist")...)
	enableTool(t, a.env.app, a.cookie, "run_command")
	writeBuild(t, a, broken)

	created := do(t, a.env.app, "POST", "/api/tasks",
		`{"projectId":"`+a.project+`","prompt":"Package it","team":true}`, jsonCookie(a.cookie))
	id, _ := created.JSON(t)["id"].(string)
	a.waitApproving(t, id, "done")

	mustContain(t, do(t, a.env.app, "GET", "/api/extensions", "", jsonCookie(a.cookie)).Body, `"extensions":[]`)

	events := string(do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(a.cookie)).Body)
	if !strings.Contains(events, "missing-popup.html") {
		t.Fatalf("the builder is told which file is missing: %s", events)
	}
}
