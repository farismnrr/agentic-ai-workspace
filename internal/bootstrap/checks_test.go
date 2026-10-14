package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodChecks = `[{"name":"hello","description":"Always passes","command":["sh","-c","exit 0"],"timeoutSeconds":30}]`

func (a agentRig) putChecks(t *testing.T, body string) reply {
	t.Helper()

	return do(t, a.env.app, "PUT", "/api/projects/"+a.project+"/checks", body, jsonCookie(a.cookie))
}

func TestChecksAreSavedReadBackAndValidated(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)

	if got := a.putChecks(t, goodChecks).Status; got != 204 {
		t.Fatalf("saving valid checks failed: %d", got)
	}

	read := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/checks", "", jsonCookie(a.cookie))
	mustContain(t, read.Body, `"name":"hello"`, `"command":["sh","-c","exit 0"]`, `"timeoutSeconds":30`)

	if _, err := os.Stat(filepath.Join(a.env.dataDir, "projects", a.project, ".workspace", "tasks.json")); err != nil {
		t.Fatalf("checks are stored in the project's folder: %v", err)
	}

	for _, bad := range []string{
		`[{"name":"Bad Name","command":["go"]}]`,
		`[{"name":"x","command":[]}]`,
		`[{"name":"x","command":["go"],"timeoutSeconds":9999}]`,
		`[{"name":"x","command":["go"]},{"name":"x","command":["go"]}]`,
	} {
		if got := a.putChecks(t, bad); got.Status != 400 || !strings.Contains(string(got.Body), "invalid_checks") {
			t.Errorf("%s must be refused, got %d %s", bad, got.Status, got.Body)
		}
	}
}

func TestChecksBelongToTheProjectOwner(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)
	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))

	if got := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/checks", "", jsonCookie(other)).Status; got != 404 {
		t.Errorf("reading someone else's checks = %d, want 404", got)
	}

	body := goodChecks

	if got := do(t, a.env.app, "PUT", "/api/projects/"+a.project+"/checks", body, jsonCookie(other)).Status; got != 404 {
		t.Errorf("writing someone else's checks = %d, want 404", got)
	}
}

func TestPassingCheckAfterTheLastChangeMakesTheWorkProven(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	a := newAgentRig(t,
		toolCall("w1", "write_file", map[string]string{"path": "page.html", "content": "<p>hi</p>"}),
		toolCall("r1", "run_task", map[string]string{"name": "hello"}),
		finishCall("Made the page and checked it"),
	)
	a.putChecks(t, goodChecks)

	id := a.start(t, "Make a page")
	a.waitApproving(t, id, "review")

	task := do(t, a.env.app, "GET", "/api/tasks/"+id, "", jsonCookie(a.cookie)).Body
	mustContain(t, task, `"proven":true`, "Ran hello", "Verified", `The \"hello\" task passed`)
}
