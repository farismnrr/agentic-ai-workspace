package connectors_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"github.com/gofiber/fiber/v3"
)

func orchestratorToken(t *testing.T, app *fiber.App, cookie, client string, extra ...string) string {
	t.Helper()
	resource := apptest.PublicURL + "/api/orchestrator"
	scopes := "orchestrator:read orchestrator:tasks orchestrator:delegate orchestrator:files"
	if len(extra) != 0 {
		scopes += " " + extra[0]
	}
	query := url.Values{
		"response_type": {"code"}, "client_id": {client}, "redirect_uri": {apptest.Redirect},
		"scope": {scopes},
		"state": {"s1"}, "code_challenge": {apptest.PKCEChallenge},
		"code_challenge_method": {"S256"}, "resource": {resource},
	}
	auth := apptest.Do(t, app, "GET", "/oauth/authorize?"+query.Encode(), "", nil)
	loc, err := url.Parse(auth.Header.Get("Location"))
	if err != nil || auth.Status != 302 {
		t.Fatalf("orchestrator authorize: %d %v", auth.Status, err)
	}
	approval := apptest.Do(t, app, "POST", "/api/oauth/requests/"+loc.Query().Get("request")+"/decision",
		`{"approve":true}`, apptest.JSONCookie(cookie))
	back, err := url.Parse(approval.JSON(t)["redirect"].(string))
	if err != nil {
		t.Fatal(err)
	}
	result := apptest.Do(t, app, "POST", "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
		"client_id": {client}, "redirect_uri": {apptest.Redirect},
		"code_verifier": {apptest.PKCEVerifier}, "resource": {resource},
	}.Encode(), apptest.FormHeaders())
	token, _ := result.JSON(t)["access_token"].(string)
	if token == "" {
		t.Fatalf("orchestrator token empty: %s", result.Body)
	}
	return token
}

func orchestratorCall(t *testing.T, app *fiber.App, token, name, args string) apptest.Reply {
	t.Helper()
	return apptest.Do(t, app, "POST", "/api/orchestrator", apptest.RPC("tools/call",
		`{"name":"`+name+`","arguments":`+args+`}`), apptest.MCPHeaders(token))
}

func orchestratorOn(t *testing.T, app *fiber.App, cookie, name string) {
	t.Helper()
	r := apptest.Do(t, app, "PUT", "/api/spaces/orchestrator/tools/"+name,
		`{"enabled":true}`, apptest.JSONCookie(cookie))
	if r.Status != 204 {
		t.Fatalf("enable %s: %d %s", name, r.Status, r.Body)
	}
}

func orchestratorData(t *testing.T, r apptest.Reply) map[string]any {
	t.Helper()
	var envelope struct {
		Result struct {
			IsError bool           `json:"isError"`
			Data    map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(r.Body, &envelope); err != nil || envelope.Result.IsError {
		t.Fatalf("orchestrator call failed: %s %v", r.Body, err)
	}
	return envelope.Result.Data
}

func TestOrchestratorSpaceSeparateResourceAndOneTask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := orchestratorToken(t, env.App, cookie, client)
	metadata := apptest.Do(t, env.App, "GET", "/.well-known/oauth-protected-resource/api/orchestrator", "", nil)
	apptest.MustContain(t, metadata.Body, `"resource":"`+apptest.PublicURL+`/api/orchestrator"`)
	cross := apptest.Do(t, env.App, "POST", "/api/coding", apptest.RPC("tools/list", `{}`), apptest.MCPHeaders(token))
	if cross.Status != 401 {
		t.Fatalf("Orchestrator token crossed Coding boundary: %d", cross.Status)
	}
	tools := apptest.Do(t, env.App, "POST", "/api/orchestrator", apptest.RPC("tools/list", `{}`), apptest.MCPHeaders(token))
	apptest.MustContain(t, tools.Body, "delegate_agent", "list_projects", "start_upload")
	apptest.MustNotContain(t, tools.Body, `"name":"create_project"`, `"name":"begin_work"`)

	created := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Orchestrator project"}`, apptest.JSONCookie(cookie))
	project := created.JSON(t)["id"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "list_projects", `{}`))
	deny := orchestratorCall(t, env.App, token, "create_task", `{"project":"`+project+`","summary":"one issue","idempotency_key":"unique-one-12345"}`)
	apptest.MustContain(t, deny.Body, `"isError":true`)
	orchestratorOn(t, env.App, cookie, "create_task")
	args := `{"project":"` + project + `","summary":"one issue","idempotency_key":"unique-one-12345"}`
	first := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task", args))
	second := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task", args))
	if first["ID"] == nil || first["ID"] != second["ID"] {
		t.Fatalf("one issue must reuse primary task: %+v %+v", first, second)
	}
	tasks := apptest.Do(t, env.App, "GET", "/api/tasks", "", apptest.JSONCookie(cookie))
	var visible []map[string]any
	if err := json.Unmarshal(tasks.Body, &visible); err != nil || len(visible) != 1 {
		t.Fatalf("duplicate tasks: %s %v", tasks.Body, err)
	}
}

func TestOrchestratorFileToolsRespectProjectAndTask(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := orchestratorToken(t, env.App, cookie, client)
	created := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Artifacts"}`, apptest.JSONCookie(cookie))
	project := created.JSON(t)["id"].(string)
	for _, tool := range []string{"create_task", "save_file", "start_upload", "upload_chunk", "finish_upload"} {
		orchestratorOn(t, env.App, cookie, tool)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"save one file","idempotency_key":"file-task-12345"}`))["ID"].(string)
	wrong := orchestratorCall(t, env.App, token, "save_file", `{"project":"`+project+`","task_id":"missing","path":"hello.txt","content":"hello"}`)
	apptest.MustContain(t, wrong.Body, `"isError":true`)
	good := orchestratorCall(t, env.App, token, "save_file", `{"project":"`+project+`","task_id":"`+task+`","path":"hello.txt","content":"hello"}`)
	orchestratorData(t, good)
	file := filepath.Join(env.DataDir, "projects", project, "hello.txt")
	if data, err := os.ReadFile(file); err != nil || string(data) != "hello" {
		t.Fatalf("saved project file: %q %v", data, err)
	}
	content := []byte("%PDF-1.7\nsmall fixture")
	digest := sha256.Sum256(content)
	init := `{"project":"` + project + `","task_id":"` + task + `","name":"report.pdf","size":` +
		jsonNumber(len(content)) + `,"sha256":"` + hex.EncodeToString(digest[:]) + `"}`
	session := orchestratorData(t, orchestratorCall(t, env.App, token, "start_upload", init))
	id := session["id"].(string)
	chunk := `{"project":"` + project + `","upload_id":"` + id + `","offset":0,"data_base64":"` +
		base64.StdEncoding.EncodeToString(content) + `"}`
	orchestratorData(t, orchestratorCall(t, env.App, token, "upload_chunk", chunk))
	finished := orchestratorData(t, orchestratorCall(t, env.App, token, "finish_upload",
		`{"project":"`+project+`","upload_id":"`+id+`"}`))
	if finished["owner"] != nil || finished["path"] != "uploads/report.pdf" {
		t.Fatalf("unsafe upload metadata: %+v", finished)
	}
	bin, err := os.ReadFile(filepath.Join(env.DataDir, "projects", project, "uploads", "report.pdf"))
	if err != nil || string(bin) != string(content) {
		t.Fatalf("uploaded binary differs: %v", err)
	}
}

func TestOrchestratorDirectInternalResearcher(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	gateway := apptest.NewGateway(t, apptest.ToolCall("r1", "report", map[string]any{
		"summary":  "Research completed by the actual internal worker",
		"document": "# Research findings\\nMock source checked.",
	}))
	providerResponse := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k")
	provider, _ := providerResponse.JSON(t)["id"].(string)
	prefs := `{"defaultProviderId":"` + provider + `","defaultModel":"fake-model","rules":[]}`
	response := apptest.Do(t, env.App, "PUT", "/api/preferences", prefs, apptest.JSONCookie(cookie))
	if response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Delegate worker"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	token := orchestratorToken(t, env.App, cookie, client)
	orchestratorOn(t, env.App, cookie, "create_task")
	orchestratorOn(t, env.App, cookie, "delegate_agent")
	orchestratorOn(t, env.App, cookie, "cancel_delegation")
	orchestratorOn(t, env.App, cookie, "update_task_plan")
	args := `{"project":"` + project + `","summary":"Research the project","idempotency_key":"direct-research-12345"}`
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task", args))["ID"].(string)
	plan := `{"task_id":"` + task + `","steps":[{"id":"research","title":"Gather evidence","state":"active"}]}`
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan", plan))
	brief := `{"task_id":"` + task + `","agent_key":"researcher","brief":"Inspect the project and report the findings",` +
		`"file_scope":["docs"],"idempotency_key":"research-job-12345"}`
	job := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent", brief))
	id, _ := job["job_id"].(string)
	if id == "" {
		t.Fatalf("worker job not created: %+v", job)
	}
	var done bool
	for attempt := 0; attempt < 200; attempt++ {
		state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
			`{"task_id":"`+task+`","job_id":"`+id+`"}`))
		if state["status"] == "done" {
			done = true
			break
		}
		if state["status"] == "failed" {
			t.Fatalf("real internal worker failed: %+v", state)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !done {
		t.Fatal("internal worker never reported")
	}
	verifiedRetry := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent", brief))
	if verifiedRetry["status"] != "done" {
		t.Fatalf("terminal retry did not preserve its result: %+v", verifiedRetry)
	}
	var parametersVerified int
	if err := env.DB.QueryRow(
		`SELECT parameters_verified FROM task_delegations WHERE task_id = ? AND id = ?`, task, id,
	).Scan(&parametersVerified); err != nil || parametersVerified != 1 {
		t.Fatalf("new delegation lacks verified parameter provenance: %d %v", parametersVerified, err)
	}
	if _, err := env.DB.Exec(
		`UPDATE task_delegations SET status = 'waiting', execution_scope = '[]', parameters_verified = 0
		 WHERE task_id = ? AND id = ?`, task, id,
	); err != nil {
		t.Fatal(err)
	}
	legacyRetry := orchestratorCall(t, env.App, token, "delegate_agent", brief)
	apptest.MustContain(t, legacyRetry.Body, `"isError":true`, "LEGACY_PARAMETERS_UNKNOWN",
		"Cancel this job", "same task", "new idempotency key")
	cancel := orchestratorCall(t, env.App, token, "cancel_delegation",
		`{"task_id":"`+task+`","job_id":"`+id+`"}`)
	if state := orchestratorData(t, cancel); state["status"] != "cancelled" {
		t.Fatalf("legacy worker was not cancellable: %+v", state)
	}
	activity := orchestratorCall(t, env.App, token, "get_task_activity", `{"task_id":"`+task+`"}`)
	apptest.MustContain(t, activity.Body, "Researcher", "handoff")
	var count []map[string]any
	tasks := apptest.Do(t, env.App, "GET", "/api/tasks", "", apptest.JSONCookie(cookie))
	if err := json.Unmarshal(tasks.Body, &count); err != nil || len(count) != 1 {
		t.Fatalf("worker delegation spawned an internal lead task: %s", tasks.Body)
	}
}

func TestExternalWorkerReportSeparatesCommandSuccessFromProof(t *testing.T) {
	// This public MCP scenario checks the report wording against the raw recorded action result.
	// The isolated command's exit code is execution evidence; it is not a project check.
	apptest.RequireDocker(t)
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	gateway := apptest.NewGateway(t,
		apptest.ToolCall("command", "run_command", map[string]string{"command": "echo harmless"}),
		apptest.ToolCall("report", "report", map[string]string{"summary": "The harmless command ran."}),
	)
	providerResponse := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k")
	provider, _ := providerResponse.JSON(t)["id"].(string)
	if response := apptest.Do(t, env.App, "PUT", "/api/preferences",
		`{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`,
		apptest.JSONCookie(cookie)); response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Command evidence"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	apptest.Do(t, env.App, "PUT", "/api/spaces/coding/approvals", `{"skip":true}`, apptest.JSONCookie(cookie))
	apptest.EnableTool(t, env.App, cookie, "run_command")
	token := orchestratorToken(t, env.App, cookie, client)
	for _, name := range []string{"create_task", "delegate_agent", "update_task_plan"} {
		orchestratorOn(t, env.App, cookie, name)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Check command reporting","idempotency_key":"evidence-task-12345"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"check","title":"Check evidence","state":"active"}]}`))
	job := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"devops","brief":"Run echo harmless and report the result",`+
			`"idempotency_key":"evidence-worker-12345"}`))
	jobID := job["job_id"].(string)
	var report string
	for attempt := 0; attempt < 200; attempt++ {
		state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
			`{"task_id":"`+task+`","job_id":"`+jobID+`"}`))
		if state["status"] == "done" {
			report, _ = state["report"].(string)
			break
		}
		if state["status"] == "failed" {
			t.Fatalf("worker failed: %+v", state)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(report, "command succeeded; outcome unverified") ||
		strings.Contains(report, "failed, exit 0") {
		t.Fatalf("external report misstates command outcome: %s", report)
	}
	activity := orchestratorCall(t, env.App, token, "get_task_activity", `{"task_id":"`+task+`"}`)
	if !strings.Contains(string(activity.Body), `"exitCode":0`) {
		t.Fatalf("raw command action must retain exitCode 0: %s", activity.Body)
	}
}

func TestExternalWorkerUsesParentToolsWithoutCodingWorkSession(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	gateway := apptest.NewGateway(t,
		apptest.ToolCall("files", "list_files", map[string]string{"path": "."}),
		apptest.ToolCall("report", "report", map[string]string{
			"summary":  "Read-only project inspection completed.",
			"document": "The project directory was inspected with the provided internal tool.",
		}),
	)
	provider := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k").JSON(t)["id"].(string)
	if response := apptest.Do(t, env.App, "PUT", "/api/preferences",
		`{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`,
		apptest.JSONCookie(cookie)); response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"External worker tools"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	apptest.EnableTool(t, env.App, cookie, "list_files")
	token := orchestratorToken(t, env.App, cookie, client)
	for _, name := range []string{"create_task", "delegate_agent", "update_task_plan"} {
		orchestratorOn(t, env.App, cookie, name)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Inspect files without a Coding session",`+
			`"idempotency_key":"native-tools-task-12345"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"inspect","title":"Inspect files","state":"active"}]}`))
	job := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"researcher","brief":"List project files and report",`+
			`"idempotency_key":"native-tools-worker-12345"}`))
	jobID := job["job_id"].(string)
	var completed bool
	for attempt := 0; attempt < 200; attempt++ {
		state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
			`{"task_id":"`+task+`","job_id":"`+jobID+`"}`))
		if state["status"] == "done" {
			completed = true
			break
		}
		if state["status"] == "failed" {
			t.Fatalf("internal worker failed: %+v", state)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !completed {
		t.Fatal("internal worker did not report")
	}
	if len(gateway.Requests) != 2 {
		t.Fatalf("expected one native tool turn and one report turn, got %d model requests", len(gateway.Requests))
	}
	var system string
	for _, raw := range gateway.Sent(0) {
		message, ok := raw.(map[string]any)
		if ok && message["role"] == "system" {
			system, _ = message["content"].(string)
		}
	}
	for _, instruction := range []string{
		"Use only the internal tools offered in this task",
		"Do not call the separate Coding Agent connector, begin_work",
		"owner-managed changes in project-notes/, .workspace/, or uploads/ are not a blocker",
		"stage only exact files changed",
	} {
		if !strings.Contains(system, instruction) {
			t.Errorf("external worker system instructions missing %q", instruction)
		}
	}
	activity := orchestratorCall(t, env.App, token, "get_task_activity", `{"task_id":"`+task+`"}`)
	apptest.MustContain(t, activity.Body, "list_files", jobID)
	var tasks []map[string]any
	response := apptest.Do(t, env.App, "GET", "/api/tasks", "", apptest.JSONCookie(cookie))
	if err := json.Unmarshal(response.Body, &tasks); err != nil || len(tasks) != 1 || tasks[0]["id"] != task {
		t.Fatalf("delegated work must remain in the existing primary task: %s %v", response.Body, err)
	}
}

func TestOrchestratorCancelledWorkerStaysCancelled(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	gateway := apptest.NewGateway(t, apptest.ToolCall("r1", "report", map[string]any{
		"summary":  "A late report that must not replace cancellation",
		"document": "# Late result\nThis must not be delivered.",
	}))
	hold := make(chan struct{})
	gateway.Hold = hold
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(hold) }) }
	defer release()
	providerResponse := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k")
	provider, _ := providerResponse.JSON(t)["id"].(string)
	prefs := `{"defaultProviderId":"` + provider + `","defaultModel":"fake-model","rules":[]}`
	if response := apptest.Do(t, env.App, "PUT", "/api/preferences", prefs, apptest.JSONCookie(cookie)); response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Cancel worker"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	token := orchestratorToken(t, env.App, cookie, client)
	for _, name := range []string{"create_task", "delegate_agent", "cancel_delegation", "update_task_plan"} {
		orchestratorOn(t, env.App, cookie, name)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Cancel a worker","idempotency_key":"cancel-task-12345"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"research","title":"Research","state":"active"}]}`))
	job := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"researcher","brief":"Research the issue","idempotency_key":"cancel-job-12345"}`))
	id, _ := job["job_id"].(string)
	if id == "" {
		t.Fatalf("worker job not created: %+v", job)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gateway.Mu.Lock()
		started := len(gateway.Requests) > 0
		gateway.Mu.Unlock()
		if started {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	gateway.Mu.Lock()
	started := len(gateway.Requests) > 0
	gateway.Mu.Unlock()
	if !started {
		t.Fatal("worker never reached the held model request")
	}
	cancelArgs := `{"task_id":"` + task + `","job_id":"` + id + `"}`
	for attempt := 0; attempt < 2; attempt++ {
		cancelled := orchestratorData(t, orchestratorCall(t, env.App, token, "cancel_delegation", cancelArgs))
		if cancelled["status"] != "cancelled" {
			t.Fatalf("cancel attempt %d returned: %+v", attempt+1, cancelled)
		}
	}
	release()
	time.Sleep(50 * time.Millisecond)
	state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
		`{"task_id":"`+task+`","job_id":"`+id+`"}`))
	if state["status"] != "cancelled" || state["report"] != nil {
		t.Fatalf("late worker result overwrote cancellation: %+v", state)
	}
	activity := orchestratorCall(t, env.App, token, "get_task_activity", `{"task_id":"`+task+`"}`)
	apptest.MustNotContain(t, activity.Body, "The Researcher reported back")
}

func TestExternalLeadPASSRequiresServerRecordedVerification(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	gateway := apptest.NewGateway(t, apptest.ToolCall("report", "report", map[string]any{
		"summary": "QA reports that the checks passed.",
	}))
	provider := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k").JSON(t)["id"].(string)
	prefs := `{"defaultProviderId":"` + provider + `","defaultModel":"fake-model","rules":[]}`
	if response := apptest.Do(t, env.App, "PUT", "/api/preferences", prefs,
		apptest.JSONCookie(cookie)); response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"External completion evidence"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	token := orchestratorToken(t, env.App, cookie, client)
	for _, name := range []string{"create_task", "delegate_agent", "update_task_plan", "complete_task"} {
		orchestratorOn(t, env.App, cookie, name)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Check external completion proof",`+
			`"idempotency_key":"completion-proof-task-12345"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"review","title":"Review the result","state":"active"}]}`))
	job := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"qa","brief":"Review the change and report whether it is ready",`+
			`"idempotency_key":"completion-proof-qa-12345"}`))
	jobID := job["job_id"].(string)
	for attempt := 0; attempt < 200; attempt++ {
		state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
			`{"task_id":"`+task+`","job_id":"`+jobID+`"}`))
		if state["status"] == "done" {
			break
		}
		if state["status"] == "failed" {
			t.Fatalf("QA report failed: %+v", state)
		}
		if attempt == 199 {
			t.Fatal("QA report did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
	pass := orchestratorCall(t, env.App, token, "complete_task",
		`{"task_id":"`+task+`","summary":"PASS: QA says it is ready","outcome":"pass"}`)
	apptest.MustContain(t, pass.Body, `"isError":true`, "server-recorded successful verification")
	partial := orchestratorData(t, orchestratorCall(t, env.App, token, "complete_task",
		`{"task_id":"`+task+`","summary":"Partial: review completed, checks are unverified",`+
			`"outcome":"partial"}`))
	if partial["status"] != "done" {
		t.Fatalf("partial audit should remain completable: %+v", partial)
	}
}

func jsonNumber(n int) string { return fmt.Sprint(n) }
