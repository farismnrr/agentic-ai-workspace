package execution_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestExternalChatRouteRejectsCrossOriginPost(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	headers := apptest.JSONHeaders()
	headers["Cookie"] = apptest.JSONCookie(a.Cookie)["Cookie"]
	headers["Origin"] = "https://attacker.invalid"
	response := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/not-owned/chat", `{"text":"hello"}`, headers)
	if response.Status != 403 {
		t.Fatalf("cross-origin external chat request status=%d body=%s", response.Status, response.Body)
	}
	approval := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/not-owned/plan/approve",
		`{"revision":1}`, headers)
	if approval.Status != 403 {
		t.Fatalf("cross-origin plan approval status=%d body=%s", approval.Status, approval.Body)
	}
	if anonymous := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/not-owned/plan/approve",
		`{"revision":1}`, apptest.JSONHeaders()); anonymous.Status != 401 {
		t.Fatalf("anonymous plan approval status=%d body=%s", anonymous.Status, anonymous.Body)
	}
}

func TestTaskNeedsAModelAndStaysPrivate(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	project, _ := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Bakery"}`,
		apptest.JSONCookie(cookie)).JSON(t)["id"].(string)

	noModel := apptest.Do(t, env.App, "POST", "/api/tasks", `{"projectId":"`+project+`","prompt":"Hi"}`,
		apptest.JSONCookie(cookie))
	if noModel.Status != 409 || !strings.Contains(string(noModel.Body), "no_model") {
		t.Fatalf("without a provider the person is told to add one: %d %s", noModel.Status, noModel.Body)
	}

	unknown := apptest.Do(t, env.App, "POST", "/api/tasks", `{"projectId":"nope","prompt":"Hi"}`,
		apptest.JSONCookie(cookie))
	if unknown.Status != 400 {
		t.Fatalf("an unknown project is refused, got %d", unknown.Status)
	}
}

func TestAnotherPersonCannotSeeOrControlATask(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	id := a.Start(t, "Private work")
	a.WaitFor(t, id, "done")

	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	for _, path := range []string{"/api/tasks/" + id, "/api/tasks/" + id + "/events"} {
		if got := apptest.Do(t, a.Environment.App, "GET", path, "", apptest.JSONCookie(other)).Status; got != 404 {
			t.Errorf("GET %s by someone else = %d, want 404", path, got)
		}
	}

	if got := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/accept", "",
		apptest.JSONCookie(other)).Status; got != 404 {
		t.Errorf("accepting someone else's task = %d, want 404", got)
	}

	if got := apptest.Do(t, a.Environment.App, "GET", "/api/tasks", "", nil).Status; got != 401 {
		t.Errorf("anonymous listing = %d, want 401", got)
	}
}
