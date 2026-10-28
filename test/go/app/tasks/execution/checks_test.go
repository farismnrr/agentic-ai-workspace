package execution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

const goodChecks = `[{"name":"hello","description":"Always passes","command":["sh","-c","exit 0"],"timeoutSeconds":30}]`

func TestChecksAreSavedReadBackAndValidated(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)

	if got := a.PutChecks(t, goodChecks).Status; got != 204 {
		t.Fatalf("saving valid checks failed: %d", got)
	}

	read := apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+a.Project+"/checks", "", apptest.JSONCookie(a.Cookie))
	apptest.MustContain(t, read.Body, `"name":"hello"`, `"command":["sh","-c","exit 0"]`, `"timeoutSeconds":30`)

	if _, err := os.Stat(filepath.Join(a.Environment.DataDir, "projects", a.Project, ".workspace",
		"tasks.json")); err != nil {
		t.Fatalf("checks are stored in the project's folder: %v", err)
	}

	for _, bad := range []string{
		`[{"name":"Bad Name","command":["go"]}]`,
		`[{"name":"x","command":[]}]`,
		`[{"name":"x","command":["go"],"timeoutSeconds":9999}]`,
		`[{"name":"x","command":["go"]},{"name":"x","command":["go"]}]`,
	} {
		if got := a.PutChecks(t, bad); got.Status != 400 || !strings.Contains(string(got.Body), "invalid_checks") {
			t.Errorf("%s must be refused, got %d %s", bad, got.Status, got.Body)
		}
	}
}

func TestChecksBelongToTheProjectOwner(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	if got := apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+a.Project+"/checks", "",
		apptest.JSONCookie(other)).Status; got != 404 {
		t.Errorf("reading someone else's checks = %d, want 404", got)
	}

	body := goodChecks

	if got := apptest.Do(t, a.Environment.App, "PUT", "/api/projects/"+a.Project+"/checks", body,
		apptest.JSONCookie(other)).Status; got != 404 {
		t.Errorf("writing someone else's checks = %d, want 404", got)
	}
}

func TestPassingCheckAfterTheLastChangeMakesTheWorkProven(t *testing.T) {
	t.Parallel()
	apptest.RequireDocker(t)

	a := apptest.NewAgentRig(t,
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "page.txt", "content": "hi"}),
		apptest.ToolCall("r1", "run_task", map[string]string{"name": "hello"}),
		apptest.FinishCall("Made the page and checked it"),
	)
	// A no-op shell command is not evidence of a real test execution.
	a.PutFile(t, "go.mod", "module sample\n\ngo 1.22\n")
	a.PutFile(t, "sample_test.go", "package sample\n\nimport \"testing\"\n\n"+
		"func TestValue(t *testing.T) { if 1+1 != 2 { t.Fatal(\"bad\") } }\n")
	a.PutChecks(t, `[{"name":"hello","description":"Runs Go tests","command":["go","test","./..."],"timeoutSeconds":90}]`)

	id := a.Start(t, "Make a page")
	a.WaitApproving(t, id, "review")

	task := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "", apptest.JSONCookie(a.Cookie)).Body
	apptest.MustContain(t, task, `"proven":true`, "Ran 1 check, and it passed", "Verified", `The \"hello\" task passed`)
}
