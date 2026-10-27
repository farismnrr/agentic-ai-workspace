package bootstrap

import (
	"strings"
	"testing"
)

func TestExternalChatRouteRejectsCrossOriginPost(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t)
	headers := jsonHeaders()
	headers["Cookie"] = jsonCookie(a.cookie)["Cookie"]
	headers["Origin"] = "https://attacker.invalid"
	response := do(t, a.env.app, "POST", "/api/tasks/not-owned/chat", `{"text":"hello"}`, headers)
	if response.Status != 403 {
		t.Fatalf("cross-origin external chat request status=%d body=%s", response.Status, response.Body)
	}
	approval := do(t, a.env.app, "POST", "/api/tasks/not-owned/plan/approve",
		`{"revision":1}`, headers)
	if approval.Status != 403 {
		t.Fatalf("cross-origin plan approval status=%d body=%s", approval.Status, approval.Body)
	}
	if anonymous := do(t, a.env.app, "POST", "/api/tasks/not-owned/plan/approve",
		`{"revision":1}`, jsonHeaders()); anonymous.Status != 401 {
		t.Fatalf("anonymous plan approval status=%d body=%s", anonymous.Status, anonymous.Body)
	}
}

func TestTaskNeedsAModelAndStaysPrivate(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	project, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Bakery"}`, jsonCookie(cookie)).JSON(t)["id"].(string)

	noModel := do(t, env.app, "POST", "/api/tasks", `{"projectId":"`+project+`","prompt":"Hi"}`, jsonCookie(cookie))
	if noModel.Status != 409 || !strings.Contains(string(noModel.Body), "no_model") {
		t.Fatalf("without a provider the person is told to add one: %d %s", noModel.Status, noModel.Body)
	}

	unknown := do(t, env.app, "POST", "/api/tasks", `{"projectId":"nope","prompt":"Hi"}`, jsonCookie(cookie))
	if unknown.Status != 400 {
		t.Fatalf("an unknown project is refused, got %d", unknown.Status)
	}
}

func TestAnotherPersonCannotSeeOrControlATask(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	id := a.start(t, "Private work")
	a.waitFor(t, id, "done")

	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))

	for _, path := range []string{"/api/tasks/" + id, "/api/tasks/" + id + "/events"} {
		if got := do(t, a.env.app, "GET", path, "", jsonCookie(other)).Status; got != 404 {
			t.Errorf("GET %s by someone else = %d, want 404", path, got)
		}
	}

	if got := do(t, a.env.app, "POST", "/api/tasks/"+id+"/accept", "", jsonCookie(other)).Status; got != 404 {
		t.Errorf("accepting someone else's task = %d, want 404", got)
	}

	if got := do(t, a.env.app, "GET", "/api/tasks", "", nil).Status; got != 401 {
		t.Errorf("anonymous listing = %d, want 401", got)
	}
}
