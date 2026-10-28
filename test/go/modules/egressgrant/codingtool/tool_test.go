package codingtool_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	codingtool "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/codingtool"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	egressapptest "github.com/farismnrr/agentic-ai-workspace/test/go/apptest/egress"
)

func askFor(tool codingtool.Tool, task, input string) taskapp.ToolResult {
	return tool.Execute(context.Background(), taskapp.ToolScope{UserID: "u1", ProjectID: "p1", TaskID: task},
		taskapp.Call{ID: "c", Name: "request_egress", Input: json.RawMessage(input)})
}

func rows(t *testing.T, db *sql.DB, where string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM egress_grants `+where).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestTheToolDoesNotExistUnlessGrantsAreRunning(t *testing.T) {
	off := codingtool.New(nil)
	if len(off.Definitions()) != 0 {
		t.Fatal("the model must not be told about a tool that is off")
	}
	if result := askFor(off, "t1", `{"host":"a.example.com","reason":"x"}`); !result.IsError {
		t.Fatalf("calling it anyway must fail: %+v", result)
	}
}

func TestAnAgentCanOnlyAskAndTheScopeIsTheServers(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	runtime := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	tool := codingtool.New(runtime.Service())
	if len(tool.Definitions()) != 1 {
		t.Fatal("the request tool must be offered when grants run")
	}
	forged := `{"host":"Marimo.Example.com","reason":"open the notebook",` +
		`"project_id":"p2","task_id":"t2","status":"approved"}`
	result := askFor(tool, "t1", forged)
	if result.IsError || !strings.Contains(result.Output, "cannot approve") {
		t.Fatalf("%+v", result)
	}
	exact := "WHERE task_id = 't1' AND project_id = 'p1' AND status = 'pending' AND host = 'marimo.example.com' " +
		"AND requested_by_kind = 'agent' AND decided_by = ''"
	if rows(t, db, exact) != 1 || rows(t, db, "WHERE task_id = 't2'") != 0 {
		t.Fatal("the request must stay pending under the server's task and project")
	}
	if len(runtime.Registry().IDs()) != 0 {
		t.Fatal("nothing may be enforced before a person approves")
	}
}

func TestAskingAgainGivesTheSameAnswerAndNoNewRow(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	runtime := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	tool := codingtool.New(runtime.Service())
	input := `{"host":"marimo.example.com","reason":"x"}`
	askFor(tool, "t1", input)
	if again := askFor(tool, "t1", input); again.IsError || !strings.Contains(again.Output, "already asked") {
		t.Fatalf("%+v", again)
	}
	egressapptest.Approve(t, runtime, "t2", "other.example.com")
	result := askFor(tool, "t2", `{"host":"other.example.com","reason":"x"}`)
	if result.IsError || !strings.Contains(result.Output, "is allowed") {
		t.Fatalf("an approved host: %+v", result)
	}
	if rows(t, db, "") != 2 {
		t.Fatalf("%d rows", rows(t, db, ""))
	}
}

func TestBadInputIsRefusedAndSaysWhy(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	runtime := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	tool := codingtool.New(runtime.Service())
	for name, input := range map[string]string{
		"ip": `{"host":"127.0.0.1","reason":"x"}`, "wildcard": `{"host":"*.example.com","reason":"x"}`,
		"port": `{"host":"a.example.com:443","reason":"x"}`, "url": `{"host":"https://a.example.com/x","reason":"x"}`,
		"internal": `{"host":"db.internal","reason":"x"}`, "no reason": `{"host":"a.example.com","reason":" "}`,
		"long reason": `{"host":"a.example.com","reason":"` + strings.Repeat("x", 600) + `"}`,
		"not json":    `host=a.example.com`, "no host": `{"reason":"x"}`,
	} {
		if result := askFor(tool, "t1", input); !result.IsError {
			t.Errorf("%s was accepted: %+v", name, result)
		}
	}
	if rows(t, db, "") != 0 {
		t.Fatal("nothing may be saved")
	}
}

func TestSpamIsStopped(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	runtime := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	tool := codingtool.New(runtime.Service())
	var refused int
	for i := range 12 {
		input := `{"host":"h` + string(rune('a'+i)) + `.example.com","reason":"x"}`
		if result := askFor(tool, "t1", input); result.IsError {
			refused++
		}
	}
	if rows(t, db, "WHERE task_id = 't1'") != 5 || refused != 7 {
		t.Fatalf("%d saved, %d refused: five may wait at once", rows(t, db, "WHERE task_id = 't1'"), refused)
	}
	if result := askFor(tool, "t2", `{"host":"ok.example.com","reason":"x"}`); result.IsError {
		t.Fatalf("another task is not affected: %+v", result)
	}
}

func TestConcurrentCallsForOneHostMakeOneRequest(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	runtime := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	tool := codingtool.New(runtime.Service())
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			askFor(tool, "t1", `{"host":"same.example.com","reason":"x"}`)
		}()
	}
	group.Wait()
	if rows(t, db, "") != 1 {
		t.Fatalf("%d rows", rows(t, db, ""))
	}
}

func TestTheToolHasNoWayToDecideAndFollowsRunCommand(t *testing.T) {
	if codingdomain.SwitchOf("request_egress") != "run_command" {
		t.Fatal("asking for a website must follow the run_command switch")
	}
	for _, info := range codingdomain.Tools() {
		if info.Name == "request_egress" {
			t.Fatal("outside AI apps must never be offered this tool")
		}
	}
}
